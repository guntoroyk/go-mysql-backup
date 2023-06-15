package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/jasonlvhit/gocron"
)

func main() {
	s := gocron.NewScheduler()

	timeToExecute := os.Getenv("GO_MYSQL_TIME_TO_EXECUTE")

	if timeToExecute == "" {
		timeToExecute = "23:00"
	}

	// Run the backupDatabase function every timeToExecute.
	s.Every(1).Day().At(timeToExecute).Do(backupDatabase)

	log.Println("Starting go-mysql-backup to backup database at", timeToExecute, "every day.")

	<-s.Start()
}

func backupDatabase() {
	now := time.Now()

	dbNames := os.Getenv("GO_MYSQL_DB_NAMES")

	if dbNames == "" {
		log.Println("Error: GO_MYSQL_DB_NAMES is not set.")
		return
	}

	dbNamesArr := strings.Split(dbNames, ",")
	for _, dbName := range dbNamesArr {
		err := backupDatabaseByName(dbName, now)
		if err != nil {
			continue
		}
	}
}

func backupDatabaseByName(dbName string, now time.Time) error {
	dbHost := os.Getenv("GO_MYSQL_DB_HOST")
	dbPort := os.Getenv("GO_MYSQL_DB_PORT")
	dbUser := os.Getenv("GO_MYSQL_DB_USER")
	dbPassword := os.Getenv("GO_MYSQL_DB_PASSWORD")
	basePath := os.Getenv("GO_BASE_PATH")

	backupFileName := fmt.Sprint(basePath, "/", dbName, "-", now.Format("2006-01-02-15-04-05"), ".sql")

	log.Println("backupFileName:", backupFileName)

	cmd := exec.Command("mysqldump", "-h"+dbHost, "-P"+dbPort, "-u"+dbUser, "-p"+dbPassword, dbName, "--result-file="+backupFileName)

	log.Println("Running mysqldump for database", dbName, "to file", backupFileName, "at", now.Format("2006-01-02 15:04:05"))

	err := cmd.Run()
	if err != nil {
		log.Println("Error running mysqldump:", err)
		return err
	}

	err = uploadToCloudinary(backupFileName)
	if err != nil {
		log.Println("Error uploading to Cloudinary:", err)
		return err
	}

	// err = os.Remove(backupFileName)
	// if err != nil {
	// 	log.Println("Error removing backup file:", err)
	// 	return err
	// }

	return nil
}

func uploadToCloudinary(backupFileName string) error {
	// Add your Cloudinary product environment credentials.
	cdCloudName := os.Getenv("GO_MYSQL_CLOUDINARY_CLOUD_NAME")
	cdApiKey := os.Getenv("GO_MYSQL_CLOUDINARY_API_KEY")
	cdApiSecret := os.Getenv("GO_MYSQL_CLOUDINARY_API_SECRET")

	cld, err := cloudinary.NewFromParams(cdCloudName, cdApiKey, cdApiSecret)
	if err != nil {
		log.Println("Error creating Cloudinary client:", err)
		return err
	}

	// Upload the backup file to Cloudinary.
	resp, err := cld.Upload.Upload(context.Background(), backupFileName,
		uploader.UploadParams{
			PublicID: fmt.Sprint("backup-database/", backupFileName),
		})
	if err != nil {
		log.Println("Error uploading image:", err)
		return err
	}

	log.Println("Database uploaded successfully:")
	log.Println(resp)
	return nil
}
