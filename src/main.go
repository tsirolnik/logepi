package main

import (
	"database/sql"
	"os"
	"strings"
	"time"

	"fmt"
	"net/http"

	"github.com/kardianos/osext"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"

	"github.com/Sirupsen/logrus"
)

const (
	configFile     = "config"
	defaultPort    = "6080"
	defaultAddress = "0.0.0.0"
)

var (
	db            *sql.DB
	config        *viper.Viper
	serverPort    string
	serverAddress string
)

func initDBConnection(user, password, dbname, host, port, sslmode string) (*sql.DB, error) {
	connString := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		user,
		password,
		dbname,
		host,
		port,
		sslmode,
	)
	logrus.Debugf("Database connection string: %s", connString)
	db, _ := sql.Open("postgres", connString)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func pong(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "pong")
}

func log(w http.ResponseWriter, r *http.Request) {
	logrus.WithFields(logrus.Fields{
		"Time":       time.Now().String(),
		"IP":         r.RemoteAddr,
		"User-Agent": r.UserAgent(),
	}).Infof("Log access")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("ERROR|Please use POST request"))
		return
	}

	logInto := r.URL.Path[len("/log/"):]
	var keys, positions []string
	var values []interface{}
	index := 1
	if err := r.ParseForm(); err != nil {
		logrus.WithFields(logrus.Fields{
			"IP":    r.RemoteAddr,
			"Error": err.Error(),
		}).Error("Malformed POST request.")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("ERROR|%s", err.Error())))
		return
	}
	for k, v := range r.PostForm {
		keys = append(keys, k)
		values = append(values, v[0])
		positions = append(positions, fmt.Sprintf("$%d", index))
		index++
	}

	if len(keys) == 0 {
		logrus.WithFields(logrus.Fields{
			"IP": r.RemoteAddr,
		}).Error("Empty request recieved.")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("ERROR|Empty request"))
		return
	}

	insertStatement := fmt.Sprintf(
		"INSERT INTO %s (created_at, %s) VALUES(now(), %s) RETURNING *",
		logInto,
		strings.Join(keys, ","),
		strings.Join(positions, ","),
	)

	_, err := db.Query(insertStatement, values...)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"IP":    r.RemoteAddr,
			"Query": insertStatement,
			"Error": err.Error(),
		}).Error("Query error")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("ERROR|%s", err.Error())))
		return
	}

	fmt.Fprint(w, "OK")
	logrus.WithFields(logrus.Fields{
		"IP":     r.RemoteAddr,
		"Query":  insertStatement,
		"Values": values,
	}).Infof("Successfuly added log entry")

	return
}

func init() {
	var err error

	currentLocation, err := osext.ExecutableFolder()
	if err != nil {
		logrus.Panicf("Failed getting current location. %s", err.Error())
	}

	config = viper.New()
	config.SetConfigName(configFile)
	config.AddConfigPath(currentLocation)
	if err := config.ReadInConfig(); err != nil {
		logrus.Panicf("Failed reading the configuration file. %s", err.Error())
	}

	config.SetDefault("address", defaultAddress)
	serverAddress = config.GetString("address")

	config.SetDefault("port", defaultPort)
	serverPort = config.GetString("port")
	// Check for PORT env variable, override configruation if exists
	if envPort := os.Getenv("PORT"); envPort != "" {
		serverPort = envPort
		logrus.WithFields(logrus.Fields{
			"PORT": serverPort,
		}).Debug("Using environment's PORT")
	}

	dbDetails := config.GetStringMapString("database")
	sslMode := "require"
	if mode, ok := dbDetails["sslmode"]; ok {
		sslMode = mode
	}
	if sslMode != "require" {
		logrus.Warning("Warning - Using non require sslmode for DB connection")
	}
	dbPort := "5432"
	if confPort, ok := dbDetails["port"]; ok {
		dbPort = confPort
	}
	db, err = initDBConnection(
		dbDetails["user"],
		dbDetails["password"],
		dbDetails["database"],
		dbDetails["host"],
		dbPort,
		sslMode,
	)
	if err != nil {
		logrus.Fatalf("Failed connection to DB: %s", err.Error())
	}
}

func main() {

	http.HandleFunc("/ping", pong)
	http.HandleFunc("/log/", log)
	listenOn := fmt.Sprintf("%s:%s", serverAddress, serverPort)
	logrus.Info("Started Logepi on ", listenOn)
	if err := http.ListenAndServe(listenOn, nil); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Fatal("HTTP Server error")
	}
}
