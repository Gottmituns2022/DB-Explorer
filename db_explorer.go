package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type CR map[string]interface{}

type DbExplorer struct {
	DB *sql.DB
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	return DbExplorer{db}, nil
}

func (h DbExplorer) DBInnitialization(w http.ResponseWriter) map[string]bool {
	rows, err := h.DB.Query("SHOW TABLES")
	__err_500(err, w)
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		__err_500(err, w)

		tables[tableName] = true
	}

	return tables
}

func (h DbExplorer) IdNameInnitialization(tableName string, w http.ResponseWriter) string {
	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT 1", tableName)
	rows, err := h.DB.Query(query)
	__err_500(err, w)
	defer rows.Close()

	stmt, err := h.DB.Prepare("SELECT EXTRA FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? AND COLUMN_NAME = ?")
	__err_500(err, w)
	defer stmt.Close()
	columnTypes, _ := rows.ColumnTypes()
	var extra string
	var idName string
	for _, col := range columnTypes {
		colName := col.Name()
		err = stmt.QueryRow(tableName, colName).Scan(&extra)
		__err_500(err, w)
		if strings.Contains(extra, "auto_increment") {
			idName = colName
		}
	}

	return idName
}

func (h DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	tables := h.DBInnitialization(w)

	switch r.Method {
	case http.MethodGet:
		if r.URL.Path == "/" {
			h.ShowTables(w, r)
		}
		if len(parts) == 1 && parts[len(parts)-1] != "" {
			tableName := parts[0]
			if tables[tableName] {
				h.ShowRecords(w, r, tableName)
			} else {
				w.WriteHeader(http.StatusNotFound)
				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "unknown table",
				})
				__err_500(err, w)
			}
		}
		if len(parts) == 2 {
			tableName := parts[0]
			strId := parts[len(parts)-1]
			recordId, err := strconv.Atoi(strId)
			if tables[tableName] && err == nil {
				idName := h.IdNameInnitialization(tableName, w)
				h.ShowRecordInfo(w, r, tableName, recordId, idName)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	case http.MethodPut:
		tableName := parts[0]
		if tables[tableName] {
			idName := h.IdNameInnitialization(tableName, w)
			h.AddRecord(w, r, tableName, idName)
		}
	case http.MethodPost:
		tableName := parts[0]
		strId := parts[len(parts)-1]
		recordId, err := strconv.Atoi(strId)
		if tables[tableName] && err == nil {
			idName := h.IdNameInnitialization(tableName, w)
			h.UpdateRecord(w, r, tableName, recordId, idName)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case http.MethodDelete:
		tableName := parts[0]
		strId := parts[len(parts)-1]
		recordId, err := strconv.Atoi(strId)
		if tables[tableName] && err == nil {
			idName := h.IdNameInnitialization(tableName, w)
			h.DeleteRecord(w, r, tableName, recordId, idName)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func (h DbExplorer) ShowTables(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SHOW TABLES")
	__err_500(err, w)
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		__err_500(err, w)

		tables = append(tables, tableName)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"response": map[string]interface{}{
			"tables": tables,
		},
	})
	__err_500(err, w)
}

func (h DbExplorer) ShowRecords(w http.ResponseWriter, r *http.Request, tableName string) {
	var limit = 5
	var offset = 0
	queryVal := r.URL.Query()

	_, err := strconv.Atoi(queryVal.Get("limit"))
	if err == nil {
		limit, _ = strconv.Atoi(queryVal.Get("limit"))
	}
	_, err = strconv.Atoi(queryVal.Get("offset"))
	if err == nil {
		offset, _ = strconv.Atoi(queryVal.Get("limit"))
	}

	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT ? OFFSET ?", tableName)
	rows, err := h.DB.Query(query, limit, offset)
	__err_500(err, w)
	defer rows.Close()

	columns, _ := rows.Columns()
	recordings := make([]interface{}, len(columns))
	recordingsPTR := make([]interface{}, len(columns))
	for i := range recordings {
		recordingsPTR[i] = &recordings[i]
	}

	result := make([]CR, 0)
	for rows.Next() {
		resultPart := CR{}
		err = rows.Scan(recordingsPTR...)
		__err_500(err, w)

		for i, record := range recordings {
			if b, ok := record.([]byte); ok {
				strRecord := string(b)
				resultPart[columns[i]] = strRecord
			} else {
				resultPart[columns[i]] = record
			}
		}
		result = append(result, resultPart)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"response": map[string]interface{}{
			"records": result,
		},
	})
	__err_500(err, w)
}

func (h DbExplorer) ShowRecordInfo(w http.ResponseWriter, r *http.Request, tableName string, recordId int, idName string) {
	query := fmt.Sprintf("SELECT * FROM `%s` WHERE `%s` = ?", tableName, idName)
	rows, _ := h.DB.Query(query, recordId)
	defer rows.Close()

	if !rows.Next() {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "record not found",
		})
		__err_500(err, w)
		return
	}
	columns, _ := rows.Columns()
	recordings := make([]interface{}, len(columns))
	recordingsPTR := make([]interface{}, len(columns))
	for i := range recordings {
		recordingsPTR[i] = &recordings[i]
	}

	result := CR{}
	err := rows.Scan(recordingsPTR...)
	__err_500(err, w)

	for i, record := range recordings {
		if b, ok := record.([]byte); ok {
			strRecord := string(b)
			result[columns[i]] = strRecord
		} else {
			result[columns[i]] = record
		}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"response": map[string]interface{}{
			"record": result,
		},
	})
	__err_500(err, w)
	rows.Close()
}

func (h DbExplorer) AddRecord(w http.ResponseWriter, r *http.Request, tableName string, idName string) {
	query := fmt.Sprintf("SELECT * FROM `%s`", tableName)
	rows, _ := h.DB.Query(query)
	columnTypes, _ := rows.ColumnTypes()
	defer rows.Close()

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	reqMap := CR{}
	err := json.Unmarshal(body, &reqMap)
	__err_500(err, w)
	for key, value := range reqMap {
		switch v := value.(type) {
		case float64:
			reqMap[key] = int(v)
		default:
		}
	}

	recordValues := make([]interface{}, 0)
	exec := fmt.Sprintf("INSERT INTO `%s` (", tableName)
	w.Header().Set("Content-Type", "application/json")
	stmt, err := h.DB.Prepare("SELECT EXTRA, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? AND COLUMN_NAME = ?")
	defer stmt.Close()

	for _, col := range columnTypes {
		colName := col.Name()
		var colType, extra string
		err = stmt.QueryRow(tableName, colName).Scan(&extra, &colType)
		__err_500(err, w)
		if strings.Contains(extra, "auto_increment") {
			continue
		}

		recordValue := reqMap[colName]
		typeOfValue := reflect.TypeOf(recordValue)
		colNullable, _ := col.Nullable()

		if recordValue == nil {
			if colNullable {
				continue
			} else {
				recordValue = ""
				typeOfValue = reflect.TypeOf(recordValue)
			}
		}

		//fmt.Printf("%v: colType - %v recordValueType - %v\n", colName, colType, reflect.TypeOf(recordValue))
		//(typeOfValue == reflect.TypeOf(0) && colType == "int")
		if !(typeOfValue == reflect.TypeOf("") && (colType == "text" || colType == "varchar")) {
			err = json.NewEncoder(w).Encode(map[string]interface{}{
				"response": map[string]interface{}{
					"error": fmt.Sprintf("field %s have invalid type", colName),
				},
			})
			__err_500(err, w)
			return
		}
		recordValues = append(recordValues, recordValue)
		exec += fmt.Sprintf("%s, ", colName)
	}

	exec = exec[:len(exec)-2] + ") VALUES (" + strings.Repeat("?, ", len(recordValues)-1) + "?)"

	result, err := h.DB.Exec(exec, recordValues...)
	if err != nil {
		fmt.Printf("Error inserting record: %v\n", err.Error())
	}

	//affected, _ := result.RowsAffected()
	lastID, _ := result.LastInsertId()
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"response": map[string]interface{}{
			idName: lastID,
		},
	})
	__err_500(err, w)
	//fmt.Println("Insert - RowsAffected", affected, "LastInsertId: ", lastID)
}

func (h DbExplorer) UpdateRecord(w http.ResponseWriter, r *http.Request, tableName string, recordId int, idName string) {
	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT 1", tableName)
	rows, _ := h.DB.Query(query)
	columnTypes, _ := rows.ColumnTypes()
	defer rows.Close()

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	reqMap := CR{}
	err := json.Unmarshal(body, &reqMap)
	__err_500(err, w)
	for key, value := range reqMap {
		switch v := value.(type) {
		case float64:
			reqMap[key] = int(v)
		default:
		}
	}

	recordValues := make([]interface{}, 0)
	exec := fmt.Sprintf("UPDATE `%s` SET ", tableName)
	w.Header().Set("Content-Type", "application/json")
	stmt, err := h.DB.Prepare("SELECT EXTRA, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? AND COLUMN_NAME = ?")
	defer stmt.Close()

	for _, col := range columnTypes {
		colName := col.Name()
		colNullable, _ := col.Nullable()
		recordValue, exists := reqMap[colName]
		if !exists {
			continue
		}
		if recordValue == nil {
			if colNullable {
				exec += fmt.Sprintf("`%s` = NULL, ", colName)
				continue
			} else {
				w.WriteHeader(http.StatusBadRequest)
				err = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": fmt.Sprintf("field %s have invalid type", colName),
				})
				__err_500(err, w)
				return
			}
		}

		var colType, extra string
		err = stmt.QueryRow(tableName, colName).Scan(&extra, &colType)
		__err_500(err, w)
		if strings.Contains(extra, "auto_increment") && recordValue != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": fmt.Sprintf("field %s have invalid type", colName),
			})
			return
		}

		typeOfValue := reflect.TypeOf(recordValue)
		if !(typeOfValue == reflect.TypeOf("") && (colType == "text" || colType == "varchar")) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": fmt.Sprintf("field %s have invalid type", colName),
			})
			return
		}
		recordValues = append(recordValues, recordValue)
		exec += fmt.Sprintf("`%s` = ?, ", colName)
	}
	exec = exec[:len(exec)-2] + fmt.Sprintf(" WHERE `%s` = ?", idName)

	recordValues = append(recordValues, recordId)
	result, err := h.DB.Exec(exec, recordValues...)
	if err != nil {
		fmt.Printf("Error updating record: %v\n", err.Error())
	}

	affected, _ := result.RowsAffected()
	//lastID, _ := result.LastInsertId()
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"response": map[string]interface{}{
			"updated": affected,
		},
	})
	__err_500(err, w)
	//fmt.Println("Updated - RowsAffected", affected, "LastInsertId: ", lastID)
}

func (h DbExplorer) DeleteRecord(w http.ResponseWriter, r *http.Request, tableName string, recordId int, idName string) {
	exec := fmt.Sprintf("DELETE FROM `%s` WHERE `%s` = ?", tableName, idName)
	result, err := h.DB.Exec(exec, recordId)
	if err != nil {
		fmt.Printf("Error deleting record: %v\n", err.Error())
		return
	}

	deleted, _ := result.RowsAffected()
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"response": map[string]interface{}{
			"deleted": deleted,
		},
	})
	__err_500(err, w)
	//fmt.Println("Insert - RowsAffected", deleted)
}

func __err_500(err error, w http.ResponseWriter) {
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("NOT SUCCESSFUL:%v\n", err.Error())
	}
}
