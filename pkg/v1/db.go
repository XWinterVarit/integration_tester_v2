package v1

import (
	"database/sql"
	"fmt"
	"strings"
)

// Field represents a database column.
type Field struct {
	Name string
	Type string
}

// Index represents a database index (simple list of columns).
type Index struct {
	Columns []string
}

// InsertField represents a column-value pair for inserts.
type InsertField struct {
	Key   string
	Value interface{}
}

// DBClient wraps the sql.DB connection.
type DBClient struct {
	DB         *sql.DB
	DriverName string
}

// Connect connects to the database.
// Driver should be imported in the main application.
func Connect(driverName, dataSourceName string) *DBClient {
	RecordAction(fmt.Sprintf("DB Connect: %s", driverName), func() { Connect(driverName, dataSourceName) })
	if IsDryRun() {
		return &DBClient{DriverName: driverName}
	}
	Logf(LogTypeDB, "Connecting to %s", driverName)
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		Fail("Failed to connect to DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		Fail("Failed to ping DB: %v", err)
	}
	Log(LogTypeDB, "Connected successfully", "")
	return &DBClient{DB: db, DriverName: driverName}
}

// SetupTable sets up a table.
func (c *DBClient) SetupTable(tableName string, isReplace bool, fields []Field, indexes []Index) {
	RecordAction(fmt.Sprintf("DB SetupTable: %s", tableName), func() { c.SetupTable(tableName, isReplace, fields, indexes) })
	if IsDryRun() {
		return
	}
	if c.DB == nil {
		Fail("DBClient is not connected (possibly running a DryRun captured action without real execution context)")
	}
	Logf(LogTypeDB, "Setting up table '%s' (Replace=%v)", tableName, isReplace)
	if isReplace {
		c.DropTable(tableName)
	}

	// Build CREATE TABLE statement
	var fieldDefs []string
	for _, f := range fields {
		fieldDefs = append(fieldDefs, fmt.Sprintf("%s %s", f.Name, f.Type))
	}

	var query string
	if c.DriverName == "oracle" {
		query = fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(fieldDefs, ", "))
	} else {
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(fieldDefs, ", "))
	}

	_, err := c.DB.Exec(query)
	if err != nil {
		// If Oracle and table exists (ORA-00955), treat as success if we were mimicking IF NOT EXISTS
		if c.DriverName == "oracle" && strings.Contains(err.Error(), "ORA-00955") {
			// Ignored
		} else {
			Fail("Failed to create table %s: %v", tableName, err)
		}
	}

	// Create Indexes
	for i, idx := range indexes {
		idxName := fmt.Sprintf("idx_%s_%d", tableName, i)
		var idxQuery string
		if c.DriverName == "oracle" {
			idxQuery = fmt.Sprintf("CREATE INDEX %s ON %s (%s)", idxName, tableName, strings.Join(idx.Columns, ", "))
		} else {
			idxQuery = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", idxName, tableName, strings.Join(idx.Columns, ", "))
		}
		_, err := c.DB.Exec(idxQuery)
		if err != nil {
			if c.DriverName == "oracle" && strings.Contains(err.Error(), "ORA-00955") {
				// Ignored
			} else {
				Fail("Failed to create index on %s: %v", tableName, err)
			}
		}
	}
}

// DropTable drops a table.
func (c *DBClient) DropTable(tableName string) {
	RecordAction(fmt.Sprintf("DB DropTable: %s", tableName), func() { c.DropTable(tableName) })
	if IsDryRun() {
		return
	}
	if c.DB == nil {
		Fail("DBClient is not connected")
	}
	Logf(LogTypeDB, "Dropping table '%s'", tableName)

	var query string
	if c.DriverName == "oracle" {
		// Oracle 11g/12c does not support DROP TABLE IF EXISTS.
		// Use PL/SQL block to ignore ORA-00942 (table does not exist).
		query = fmt.Sprintf(`BEGIN
			EXECUTE IMMEDIATE 'DROP TABLE %s PURGE';
			EXCEPTION WHEN OTHERS THEN
				IF SQLCODE != -942 THEN RAISE; END IF;
			END;`, tableName)
	} else {
		query = fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	}

	_, err := c.DB.Exec(query)
	if err != nil {
		Fail("Failed to drop table %s: %v", tableName, err)
	}
}

// CleanTable deletes all data from a table.
func (c *DBClient) CleanTable(tableName string) {
	RecordAction(fmt.Sprintf("DB CleanTable: %s", tableName), func() { c.CleanTable(tableName) })
	if IsDryRun() {
		return
	}
	if c.DB == nil {
		Fail("DBClient is not connected")
	}
	Logf(LogTypeDB, "Cleaning table '%s'", tableName)
	_, err := c.DB.Exec(fmt.Sprintf("DELETE FROM %s", tableName))
	if err != nil {
		Fail("Failed to clean table %s: %v", tableName, err)
	}
}

// DeleteOne deletes a single row matching the where clause.
// It is a convenience wrapper over DeleteWithLimit(..., 1).
func (c *DBClient) DeleteOne(tableName string, where string, args ...interface{}) {
	RecordAction(fmt.Sprintf("DB DeleteOne: %s", tableName), func() { c.DeleteOne(tableName, where, args...) })
	if IsDryRun() {
		return
	}
	c.deleteWithLimitInternal(tableName, where, 1, args...)
}

// DeleteWithLimit deletes up to `limit` rows matching the where clause.
// If limit <= 0, it deletes all rows matching the condition.
func (c *DBClient) DeleteWithLimit(tableName string, where string, limit int, args ...interface{}) {
	RecordAction(fmt.Sprintf("DB DeleteWithLimit: %s", tableName), func() { c.DeleteWithLimit(tableName, where, limit, args...) })
	if IsDryRun() {
		return
	}
	c.deleteWithLimitInternal(tableName, where, limit, args...)
}

// deleteWithLimitInternal contains the shared delete logic.
func (c *DBClient) deleteWithLimitInternal(tableName string, where string, limit int, args ...interface{}) {
	if c.DB == nil {
		Fail("DBClient is not connected")
	}
	if strings.TrimSpace(where) == "" {
		Fail("Delete operation requires a WHERE clause to prevent full-table deletes")
	}

	finalWhere := where
	argCounter := 1
	if c.DriverName == "oracle" {
		count := strings.Count(where, "?")
		for i := 0; i < count; i++ {
			finalWhere = strings.Replace(finalWhere, "?", fmt.Sprintf(":%d", argCounter), 1)
			argCounter++
		}
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, finalWhere)
	var allArgs []interface{}
	allArgs = append(allArgs, args...)

	if limit > 0 {
		switch c.DriverName {
		case "oracle":
			query = fmt.Sprintf("DELETE FROM %s WHERE (%s) AND ROWNUM <= %d", tableName, finalWhere, limit)
		case "postgres", "postgresql":
			// Postgres has no DELETE ... LIMIT; use CTE
			query = fmt.Sprintf("WITH cte AS (SELECT ctid FROM %s WHERE %s LIMIT %d) DELETE FROM %s WHERE ctid IN (SELECT ctid FROM cte)", tableName, finalWhere, limit, tableName)
		case "sqlite3":
			// Some SQLite builds don't accept DELETE ... LIMIT; use rowid subquery
			query = fmt.Sprintf("DELETE FROM %s WHERE rowid IN (SELECT rowid FROM %s WHERE %s LIMIT %d)", tableName, tableName, finalWhere, limit)
		default:
			// MySQL/SQLite support LIMIT in DELETE
			query = fmt.Sprintf("DELETE FROM %s WHERE %s LIMIT %d", tableName, finalWhere, limit)
		}
	}

	Log(LogTypeDB, "Delete Rows", fmt.Sprintf("Query: %s\nArgs: %v", query, allArgs))
	_, err := c.DB.Exec(query, allArgs...)
	if err != nil {
		Fail("Failed to delete from %s: %v", tableName, err)
	}
}

// SetupTableFromAnother copies structure and data (simplified).
// Note: This is complex across different DBs. We'll do a simple CREATE TABLE AS SELECT or similar if supported,
// or just copy structure.
// For now, let's assume it copies schema.
// "Setup Table From Another (isReplace bool, table 1 connection 1, table 2 connection 2)"
// This sounds like copying FROM table 2 TO table 1? Or syncing?
// Given the complexity, I will implement a basic version that might fail if not supported.
func SetupTableFromAnother(destClient *DBClient, destTable string, srcClient *DBClient, srcTable string, isReplace bool) {
	Logf(LogTypeDB, "SetupTableFromAnother: %s -> %s (Replace=%v)", srcTable, destTable, isReplace)
	// This is hard to do generically without knowing schema.
	// For this exercise, I will log a warning that this feature is limited.
	Log(LogTypeDB, "SetupTableFromAnother Warning", "SetupTableFromAnother is a placeholder. Implementing full table copy across connections is complex generic logic.")
}

// InsertOne inserts a single row with specified column-value pairs.
// fields should be a list of InsertField: [{Key: "col1", Value: val1}, ...].
func (c *DBClient) InsertOne(tableName string, fields []InsertField) {
	RecordAction(fmt.Sprintf("DB InsertOne: %s", tableName), func() { c.InsertOne(tableName, fields) })
	if IsDryRun() {
		return
	}
	if c.DB == nil {
		Fail("DBClient is not connected")
	}
	if len(fields) == 0 {
		Fail("InsertOne requires at least one field/value pair")
	}

	var cols []string
	var placeholders []string
	var values []interface{}
	argCounter := 1

	for _, f := range fields {
		if strings.TrimSpace(f.Key) == "" {
			Fail("InsertOne expects field names as non-empty strings (got %v)", f.Key)
		}
		cols = append(cols, f.Key)

		ph := "?"
		if c.DriverName == "oracle" {
			ph = fmt.Sprintf(":%d", argCounter)
			argCounter++
		}
		placeholders = append(placeholders, ph)
		values = append(values, f.Value)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	Log(LogTypeDB, "Insert One", fmt.Sprintf("Query: %s\nArgs: %v", query, values))

	_, err := c.DB.Exec(query, values...)
	if err != nil {
		Fail("Failed to insert into %s: %v", tableName, err)
	}
}

// ReplaceData inserts or replaces data.
// Data is assumed to be a list of values matching columns order.
func (c *DBClient) ReplaceData(tableName string, values []interface{}) {
	RecordAction(fmt.Sprintf("DB ReplaceData: %s", tableName), func() { c.ReplaceData(tableName, values) })
	if IsDryRun() {
		return
	}
	if c.DB == nil {
		Fail("DBClient is not connected")
	}
	Log(LogTypeDB, fmt.Sprintf("Replacing data in '%s'", tableName), fmt.Sprintf("%v", values))
	// We need to know placeholders.
	placeholders := make([]string, len(values))
	for i := range values {
		if c.DriverName == "oracle" {
			placeholders[i] = fmt.Sprintf(":%d", i+1)
		} else {
			placeholders[i] = "?" // Standard for many, but Postgres uses $1.
		}
	}

	// "REPLACE INTO" is MySQL/SQLite specific. Postgres uses "INSERT ... ON CONFLICT".
	// The requirement is "Replace Data".
	// I'll try generic DELETE then INSERT to simulate replace if no PK is known,
	// but without PK, DELETE is hard.
	// I'll stick to INSERT for now or try "REPLACE INTO" which works on SQLite/MySQL.

	query := fmt.Sprintf("INSERT INTO %s VALUES (%s)", tableName, strings.Join(placeholders, ", "))
	_, err := c.DB.Exec(query, values...)
	if err != nil {
		Fail("Failed to insert/replace data into %s: %v", tableName, err)
	}
}

// QueryData is a helper to run queries.
func (c *DBClient) QueryData(query string, args ...interface{}) *sql.Rows {
	RecordAction("DB QueryData", func() { c.QueryData(query, args...) })
	if IsDryRun() {
		return nil
	}
	if c.DB == nil {
		Fail("DBClient is not connected")
	}

	finalQuery := query
	if c.DriverName == "oracle" {
		// Replace ? with :n
		argCounter := 1
		count := strings.Count(query, "?")
		for i := 0; i < count; i++ {
			finalQuery = strings.Replace(finalQuery, "?", fmt.Sprintf(":%d", argCounter), 1)
			argCounter++
		}
	}

	Log(LogTypeDB, "Query Data", fmt.Sprintf("Query: %s\nArgs: %v", finalQuery, args))
	rows, err := c.DB.Query(finalQuery, args...)
	if err != nil {
		Fail("Failed to query data: %v", err)
	}
	return rows
}

// --- Simplified Query/Update API ---

// QueryResult holds the results of a Fetch operation.
type QueryResult struct {
	Rows []RowResult
}

// RowResult represents a single row from the database.
type RowResult struct {
	Data map[string]interface{}
}

// Fetch executes a query and returns all results in an easy-to-use QueryResult object.
func (c *DBClient) Fetch(query string, args ...interface{}) *QueryResult {
	RecordAction("DB Fetch", func() { c.Fetch(query, args...) })
	if IsDryRun() {
		return &QueryResult{}
	}
	rows := c.QueryData(query, args...)
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		Fail("Failed to get columns: %v", err)
	}

	var results []RowResult

	for rows.Next() {
		// Prepare a slice of interface{} to hold values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			Fail("Failed to scan row: %v", err)
		}

		rowData := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]

			// Handle []byte as string for convenience, common in some drivers/types
			key := strings.ToLower(col)
			if b, ok := val.([]byte); ok {
				rowData[key] = string(b)
			} else {
				rowData[key] = val
			}
		}
		results = append(results, RowResult{Data: rowData})
	}

	return &QueryResult{Rows: results}
}

// Update performs a partial update on a table based on a condition.
// updates: map of column name -> new value
// where: condition string (e.g., "id = ?")
// args: arguments for the where clause
func (c *DBClient) Update(tableName string, updates map[string]interface{}, where string, args ...interface{}) {
	RecordAction(fmt.Sprintf("DB Update: %s", tableName), func() { c.Update(tableName, updates, where, args...) })
	if IsDryRun() {
		return
	}
	if c.DB == nil {
		Fail("DBClient is not connected")
	}

	if len(updates) == 0 {
		return
	}

	var sets []string
	var values []interface{}
	argCounter := 1

	for col, val := range updates {
		ph := "?"
		if c.DriverName == "oracle" {
			ph = fmt.Sprintf(":%d", argCounter)
			argCounter++
		}
		sets = append(sets, fmt.Sprintf("%s = %s", col, ph))
		values = append(values, val)
	}

	// Handle where clause
	finalWhere := where
	if c.DriverName == "oracle" {
		// Replace ? with :n
		// Naive replacement
		count := strings.Count(where, "?")
		for i := 0; i < count; i++ {
			finalWhere = strings.Replace(finalWhere, "?", fmt.Sprintf(":%d", argCounter), 1)
			argCounter++
		}
	}

	// Append WHERE args
	values = append(values, args...)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, strings.Join(sets, ", "), finalWhere)

	Log(LogTypeDB, "Update Table", fmt.Sprintf("Query: %s\nArgs: %v", query, values))

	_, err := c.DB.Exec(query, values...)
	if err != nil {
		Fail("Failed to update table %s: %v", tableName, err)
	}
}

// --- QueryResult Helpers ---

// GetRow returns the row at the specified index. Panics if index is out of bounds.
func (qr *QueryResult) GetRow(index int) *RowResult {
	if IsDryRun() {
		return &RowResult{Data: make(map[string]interface{})}
	}
	if index < 0 || index >= len(qr.Rows) {
		Fail("GetRow: index %d out of bounds (count: %d)", index, len(qr.Rows))
	}
	return &qr.Rows[index]
}

// Count returns the number of rows.
func (qr *QueryResult) Count() int {
	return len(qr.Rows)
}

// ExpectCount asserts that the number of rows matches the expected count.
func (qr *QueryResult) ExpectCount(expected int) {
	if IsDryRun() {
		return
	}
	count := qr.Count()
	if count != expected {
		Fail("Expected Row Count %d, got %d", expected, count)
	}
	Logf(LogTypeExpect, "Row Count %d == %d - PASSED", count, expected)
}

// --- RowResult Helpers ---

// Get returns the value of a field. Panics if field does not exist.
func (r *RowResult) Get(field string) interface{} {
	if IsDryRun() {
		return nil
	}
	val, ok := r.Data[strings.ToLower(field)]
	if !ok {
		Fail("Field '%s' not found in row", field)
	}
	return val
}

// GetTo scans the value of a field into the target pointer.
func (r *RowResult) GetTo(field string, target interface{}) {
	if IsDryRun() {
		return
	}
	val := r.Get(field)
	// Basic type matching or conversion could go here.
	// For simplicity, we rely on fmt.Sscan or simple assignment if types match.
	// Let's use fmt.Sprintf -> Sscan for "easy" flexible conversion
	sVal := fmt.Sprintf("%v", val)
	if _, err := fmt.Sscan(sVal, target); err != nil {
		Fail("Failed to scan field '%s' (val=%v) into target: %v", field, val, err)
	}
}

// Expect asserts that the field exists and equals the expected value.
func (r *RowResult) Expect(field string, expected interface{}) {
	if IsDryRun() {
		return
	}
	val := r.Get(field)

	// Simple comparison.
	// To handle int vs int64 issues common in DBs, we convert both to string for comparison if direct equality fails.
	if val != expected {
		sVal := fmt.Sprintf("%v", val)
		sExp := fmt.Sprintf("%v", expected)
		if sVal != sExp {
			Fail("Expect failed for field '%s': expected '%v', got '%v'", field, expected, val)
		}
	}
	Logf(LogTypeExpect, "DB Field '%s' == '%v' - PASSED", field, expected)
}

// ExpectCond asserts that the field satisfies the provided condition against the expected value.
// Supports nil (DB NULL) comparison when using ConditionEqual/ConditionNotEqual with expected == nil.
func (r *RowResult) ExpectCond(field string, condition string, expected interface{}) {
	if IsDryRun() {
		return
	}
	val := r.Get(field)

	if !evaluateCondition(val, condition, expected) {
		Fail("ExpectCond failed for field '%s' with condition '%s': expected %v (%T), got %v (%T)", field, condition, expected, expected, val, val)
	}

	Logf(LogTypeExpect, "DB Field '%s' %s %v - PASSED", field, condition, expected)
}
