package restapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// FileRequest represents the structure of the incoming request
type FileRequest struct {
	Filename string `json:"filename,omitempty"`
	Text     string `json:"text,omitempty"`
}

// FileResponse represents the structure of the outgoing response
type FileResponse struct {
	Filename string `json:"filename"`
	Message  string `json:"message"`
}

func generateFilename() string {
	now := time.Now()
	return fmt.Sprintf("%02d%02d%02d%03d%02d%02d%04d.tst",
		now.Hour(), now.Minute(), now.Second(), now.Nanosecond()/1e6, // Convert nanoseconds to milliseconds
		now.Month(), now.Day(), now.Year())
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	var req FileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Generate filename if not provided

	// Connect to the SFTP server and transfer the file
	req, err := transferFileSFTP(req)
	if err != nil {
		log.Printf("Failed to transfer file via SFTP: %v", err)
		err_msg := fmt.Errorf("failed to fetch local directory from Redis: %v", err)
		http.Error(w, err_msg.Error(), http.StatusInternalServerError)
		return
	}

	response := FileResponse{
		Filename: req.Filename,
		Message:  "File transferred successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func transferFileSFTP(req FileRequest) (FileRequest, error) {

	if req.Filename == "" {
		req.Filename = generateFilename()
	}
	// Fetch local directory from Redis
	localDir, err := rdb.HGet(ctx, "ENV_VARIABLES_TC_APP", "LOCAL_TC_FILES_DIRECTORY").Result()
	if err != nil {
		localDir = "C:/Users/Public/TC_SENT_FILES"
		rdb.HSet(ctx, "ENV_VARIABLES_TC_APP", "LOCAL_TC_FILES_DIRECTORY", "C:/Users/Public/TC_SENT_FILES").Result()
		return req, fmt.Errorf("failed to fetch local directory from Redis: %v", err)
	}

	// Create the local directory if it does not exist
	if err := os.MkdirAll(localDir, os.ModePerm); err != nil {
		return req, fmt.Errorf("failed to create local directory: %v", err)
	}

	// Create the file in the local directory
	localFilePath := filepath.Join(localDir, req.Filename)
	err = os.WriteFile(localFilePath, []byte(req.Text), 0644)
	if err != nil {
		return req, fmt.Errorf("failed to write file: %v", err)
	}

	// Fetch SFTP details from Redis
	sftpUser, err := rdb.HGet(ctx, "ENV_VARIABLES_UMACS", "SFTP_USER").Result()
	if err != nil {
		rdb.HSet(ctx, "ENV_VARIABLES_UMACS", "SFTP_USER", "SFTP_USER").Result()
		return req, fmt.Errorf("failed to fetch SFTP user from Redis: %v", err)

	}

	sftpPass, err := rdb.HGet(ctx, "ENV_VARIABLES_UMACS", "SFTP_PASSWORD").Result()
	if err != nil {
		rdb.HSet(ctx, "ENV_VARIABLES_UMACS", "SFTP_PASSWORD", "SFTP_PASSWORD").Result()
		return req, fmt.Errorf("failed to fetch SFTP password from Redis: %v", err)
	}

	sftpHost, err := rdb.HGet(ctx, "ENV_VARIABLES_UMACS", "SERVER_IP").Result()
	if err != nil {
		rdb.HSet(ctx, "ENV_VARIABLES_UMACS", "SERVER_IP", "172.20.5.xx").Result()
		return req, fmt.Errorf("failed to fetch SFTP host from Redis: %v", err)
	}

	sftpPort, err := rdb.HGet(ctx, "ENV_VARIABLES_UMACS", "SFTP_PORT").Result()
	if err != nil {
		sftpPort = "22" // Default SFTP port if not provided in Redis
		rdb.HSet(ctx, "ENV_VARIABLES_UMACS", "SFTP_PORT", 22).Result()
	}

	if _, err := strconv.Atoi(sftpPort); err != nil {
		return req, fmt.Errorf("invalid SFTP port: %v Err: %v", sftpPort, err)
	}

	sftpRemoteDir, err := rdb.HGet(ctx, "ENV_VARIABLES_UMACS", "UMACS_TC_FILE_DIR").Result()
	if err != nil {
		rdb.HSet(ctx, "ENV_VARIABLES_UMACS", "UMACS_TC_FILE_DIR", "/umacsops/exe").Result()
		log.Printf("Failed to fetch SFTP remote directory from Redis: %v", err)
		return req, fmt.Errorf("failed to fetch SFTP remote directory from Redis: %v", err)
	}

	config := &ssh.ClientConfig{
		User: sftpUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(sftpPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", sftpHost, sftpPort)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return req, fmt.Errorf("failed to dial SSH: %v", err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return req, fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer client.Close()

	srcFile, err := os.Open(localFilePath)
	if err != nil {
		return req, fmt.Errorf("failed to open local file: %v", err)
	}
	defer srcFile.Close()

	remoteFilePath := filepath.Join(sftpRemoteDir, filepath.Base(localFilePath))
	dstFile, err := client.Create(remoteFilePath)
	if err != nil {
		return req, fmt.Errorf("failed to create remote file: %v", err)
	}
	defer dstFile.Close()

	_, err = srcFile.Seek(0, 0)
	if err != nil {
		return req, fmt.Errorf("failed to seek local file: %v", err)
	}

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return req, fmt.Errorf("failed to copy file content: %v", err)
	}

	return req, nil
}

func registerRoutesForFileTransferSftp(r *mux.Router) {
	r.HandleFunc("/file_transfer_sftp", handleFileUpload).Methods("POST")
}
