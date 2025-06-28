package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var db *sql.DB
var deptShortToID map[string]int // Cache for department short name to ID mapping

// Department struct for m_departments table
type Department struct {
	ID        int    `json:"id"`
	DeptShort string `json:"dept_short"`
	DeptName  string `json:"dept_name"`
}

// User struct defines the data model for a user from m_users.
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Dept     int    `json:"dept"` // Stored as integer ID
	DeptName string `json:"deptName"` // Fetched via JOIN for display
	Year     string `json:"year"`
	Degree   string `json:"degree"`
}

// init function connects to DB and populates the department map on startup.
func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, assuming environment variables are set.")
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	if dbUser == "" || dbPassword == "" || dbHost == "" || dbPort == "" || dbName == "" {
		log.Fatal("Database environment variables not set.")
	}

	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	db, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	fmt.Println("Successfully connected to MySQL database!")

	// Populate the department map on startup
	if err := populateDeptMap(); err != nil {
		log.Fatalf("Failed to populate department map: %v", err)
	}
}

// populateDeptMap fetches all departments and populates the deptShortToID map.
func populateDeptMap() error {
	rows, err := db.Query("SELECT id, dept_short FROM m_departments WHERE status = '1'")
	if err != nil {
		return err
	}
	defer rows.Close()

	deptShortToID = make(map[string]int)
	for rows.Next() {
		var id int
		var deptShort string
		if err := rows.Scan(&id, &deptShort); err != nil {
			log.Printf("Warning: could not scan department row: %v", err)
			continue
		}
		deptShortToID[strings.ToUpper(deptShort)] = id
	}
	fmt.Printf("Successfully loaded %d departments into map.\n", len(deptShortToID))
	return rows.Err()
}

func mapDegreeToEnum(degree string) string {
	switch strings.ToUpper(strings.TrimSpace(degree)) {
	case "UG", "B.TECH", "BTECH":
		return "UG"
	case "PG", "M.TECH", "MTECH":
		return "PG"
	case "MBA":
		return "MBA"
	case "PHD":
		return "PHD"
	default:
		return "UG" // Sensible default
	}
}

func mapEnumToDegree(enumVal string) string {
	return enumVal // The enum values are user-friendly enough
}

// uploadUsers handles CSV file upload and processes records into m_users.
func uploadUsers(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No file uploaded", "error": err.Error()})
		return
	}

	// Ensure the file is a CSV
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".csv") {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid file format. Please upload a CSV file."})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to open uploaded file", "error": err.Error()})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// This will help with variable numbers of columns if your CSV has them
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse CSV file", "error": err.Error()})
		return
	}

	if len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "CSV file is empty or has only a header row."})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to start database transaction", "error": err.Error()})
		return
	}
	defer tx.Rollback() // Rollback on any error

	// Adjusted statement to only insert the columns from the CSV
	stmt, err := tx.Prepare(`
		INSERT INTO m_users (id, name, email, dept, year, degree) 
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
			name = VALUES(name), 
			email = VALUES(email), 
			dept = VALUES(dept), 
			year = VALUES(year), 
			degree = VALUES(degree),
			status = '0' -- Reset status on update if needed
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to prepare SQL statement", "error": err.Error()})
		return
	}
	defer stmt.Close()

	var processingErrors []string
	// Expecting CSV Column Order: 0:Reg No, 1:Name, 2:Email, 3:Dept, 4:Year, 5:Degree
	for i, row := range rows[1:] { // Skip header row
		if len(row) < 6 {
			processingErrors = append(processingErrors, fmt.Sprintf("Row %d: Insufficient columns. Expected 6, got %d. Skipping.", i+2, len(row)))
			continue
		}
        
        // Trim whitespace from all fields to avoid issues
        id := strings.TrimSpace(row[0])
        name := strings.TrimSpace(row[1])
        email := strings.TrimSpace(row[2])
        deptShort := strings.ToUpper(strings.TrimSpace(row[3]))
        year := strings.TrimSpace(row[4])
        degree := strings.TrimSpace(row[5])


		if id == "" {
			processingErrors = append(processingErrors, fmt.Sprintf("Row %d: 'id' (Reg No) cannot be empty. Skipping.", i+2))
			continue
		}


		deptID, ok := deptShortToID[deptShort]
		if !ok {
			processingErrors = append(processingErrors, fmt.Sprintf("Row %d (ID: %s): Department '%s' not found. Skipping.", i+2, id, row[3]))
			continue
		}

		degreeEnum := mapDegreeToEnum(degree)

		_, err = stmt.Exec(id, name, email, deptID, year, degreeEnum)
		if err != nil {
			processingErrors = append(processingErrors, fmt.Sprintf("Row %d (ID: %s): Database error: %v. Skipping.", i+2, id, err))
		}
	}

	if err := tx.Commit(); err != nil {
		// If commit fails, it's a server error. The transaction would have been rolled back by defer.
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to commit database transaction", "error": err.Error()})
		return
	}

	if len(processingErrors) > 0 {
		c.JSON(http.StatusAccepted, gin.H{"message": "Users uploaded with some errors.", "errors": processingErrors})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Users uploaded successfully."})
}


// getUsers retrieves all users, joining with departments to get dept_short.
func getUsers(c *gin.Context) {
	query := `
		SELECT u.id, u.name, u.email, u.dept, d.dept_short, u.year, u.degree 
		FROM m_users u
		LEFT JOIN m_departments d ON u.dept = d.id
		ORDER BY u.id
	`
	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch user data", "error": err.Error()})
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var degreeEnum string
		var deptName sql.NullString // Use sql.NullString for LEFT JOIN
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Dept, &deptName, &user.Year, &degreeEnum); err != nil {
			log.Printf("Error scanning user row: %v", err)
			continue
		}
		user.DeptName = deptName.String
		user.Degree = mapEnumToDegree(degreeEnum)
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error iterating through user data", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// getDepartments retrieves all active departments.
func getDepartments(c *gin.Context) {
	rows, err := db.Query("SELECT id, dept_short, dept_name FROM m_departments WHERE status = '1' ORDER BY dept_name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch department data", "error": err.Error()})
		return
	}
	defer rows.Close()

	var departments []Department
	for rows.Next() {
		var dept Department
		if err := rows.Scan(&dept.ID, &dept.DeptShort, &dept.DeptName); err != nil {
			log.Printf("Error scanning department row: %v", err)
			continue
		}
		departments = append(departments, dept)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error iterating department data", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, departments)
}

// updateUser handles updating a single user entry.
func updateUser(c *gin.Context) {
	id := c.Param("id")

	var updatedUser User
	if err := c.ShouldBindJSON(&updatedUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data provided: " + err.Error()})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction: " + err.Error()})
		return
	}
	defer tx.Rollback()

	// --- Start of added email uniqueness check ---
	var existingID string
	err = tx.QueryRow("SELECT id FROM m_users WHERE email = ? AND id != ?", updatedUser.Email, id).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error during email check: " + err.Error()})
		return
	}
	if existingID != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists for another user."})
		return
	}
	// --- End of added email uniqueness check ---

	degreeEnum := mapDegreeToEnum(updatedUser.Degree)

	stmt, err := tx.Prepare("UPDATE m_users SET name = ?, email = ?, dept = ?, year = ?, degree = ? WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare update statement: " + err.Error()})
		return
	}
	defer stmt.Close()

	// Corrected order of parameters
	_, err = stmt.Exec(updatedUser.Name, updatedUser.Email, updatedUser.Dept, updatedUser.Year, degreeEnum, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute update: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// deleteUser handles deleting a user from m_users.
func deleteUser(c *gin.Context) {
	id := c.Param("id")

	res, err := db.Exec("DELETE FROM m_users WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute delete: " + err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No user found with the given ID"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// main function sets up the Gin router and starts the server.
func main() {
	r := gin.Default()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// A more permissive CORS for local development
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.POST("/upload-users", uploadUsers)
		api.GET("/users", getUsers)
		api.PUT("/users/:id", updateUser)
		api.DELETE("/users/:id", deleteUser)
		api.GET("/departments", getDepartments)
	}

	fmt.Println("Server starting on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}