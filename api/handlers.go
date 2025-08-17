package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

type Input struct {
	Pregnancies              int     `json:"pregnancies"`
	Glucose                  float64 `json:"glucose"`
	BloodPressure            float64 `json:"bloodPressure"`
	SkinThickness            float64 `json:"skinThickness"`
	Insulin                  float64 `json:"insulin"`
	BMI                      float64 `json:"bmi"`
	DiabetesPedigreeFunction float64 `json:"diabetesPedigreeFunction"`
	Age                      int     `json:"age"`
}

func PredictHandler(c *gin.Context) {
	var input Input
	if err := c.BindJSON(&input); err != nil {
		fmt.Printf("Error binding JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := resty.New()
	resp, err := client.R().
		SetBody(input).
		SetResult(&struct {
			Prediction string `json:"prediction"`
		}{}).
		Post("https://smart-disease-predictor-ml.onrender.com/predict")
	if err != nil {
		fmt.Printf("Error connecting to ML server: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to ML server: " + err.Error()})
		return
	}
	if resp.StatusCode() != http.StatusOK {
		fmt.Printf("ML server returned non-200 status: %d, body: %s\n", resp.StatusCode(), string(resp.Body()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ML server error: status %d", resp.StatusCode())})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prediction": resp.Result().(*struct {
		Prediction string `json:"prediction"`
	}).Prediction})
}

func ExtractHandler(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		fmt.Printf("Error getting form file: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image uploaded: " + err.Error()})
		return
	}
	fmt.Printf("Received file: %s, size: %d bytes\n", file.Filename, file.Size)

	tempPath := "temp.jpg"
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		fmt.Printf("Error saving file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image: " + err.Error()})
		return
	}
	defer os.Remove(tempPath)
	fmt.Printf("Saved file to: %s\n", tempPath)

	// Verify Tesseract is accessible
	if _, err := exec.LookPath("tesseract"); err != nil {
		fmt.Printf("Tesseract not found: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tesseract not installed or not in PATH: " + err.Error()})
		return
	}

	// Run Tesseract OCR
	outputFile := "output"
	cmd := exec.Command("tesseract", tempPath, outputFile, "-l", "eng")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Tesseract execution failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCR failed: " + err.Error()})
		return
	}
	fmt.Printf("Tesseract executed successfully, output: %s.txt\n", outputFile)

	// Read the output text file
	text, err := os.ReadFile(outputFile + ".txt")
	if err != nil {
		fmt.Printf("Error reading OCR output: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read OCR output: " + err.Error()})
		return
	}
	defer os.Remove(outputFile + ".txt")

	ocrText := strings.TrimSpace(string(text))
	fmt.Printf("OCR output:\n%s\n", ocrText)

	if ocrText == "" {
		fmt.Println("OCR output is empty")
		c.JSON(http.StatusOK, gin.H{"extracted": map[string]float64{}})
		return
	}

	// Normalize common OCR variants
	ocrText = strings.ReplaceAll(ocrText, "BloodPressure", "Blood Pressure")
	ocrText = strings.ReplaceAll(ocrText, "SkinThickness", "Skin Thickness")
	ocrText = strings.ReplaceAll(ocrText, "DPF", "Diabetes Pedigree Function")

	// Regex patterns
	patterns := map[string]*regexp.Regexp{
		"Pregnancies":              regexp.MustCompile((?i)Pregnancies\s*[:=\-]?\s*(\d+)),
		"Glucose":                  regexp.MustCompile((?i)Glucose\s*[:=\-]?\s*(\d+\.?\d*)),
		"BloodPressure":            regexp.MustCompile((?i)Blood\s*Pressure\s*[:=\-]?\s*(\d+\.?\d*)),
		"SkinThickness":            regexp.MustCompile((?i)Skin\s*Thickness\s*[:=\-]?\s*(\d+\.?\d*)),
		"Insulin":                  regexp.MustCompile((?i)Insulin\s*[:=\-]?\s*(\d+\.?\d*)),
		"BMI":                      regexp.MustCompile((?i)BMI\s*[:=\-]?\s*(\d+\.?\d*)),
		"DiabetesPedigreeFunction": regexp.MustCompile((?i)Diabetes\s*Pedigree\s*Function\s*[:=\-]?\s*(\d+\.?\d*)),
		"Age":                      regexp.MustCompile((?i)Age\s*[:=\-]?\s*(\d+)),
	}

	// Extract values
	extracted := make(map[string]float64)
	for key, re := range patterns {
		if match := re.FindStringSubmatch(ocrText); len(match) > 1 {
			if f, err := strconv.ParseFloat(match[1], 64); err == nil {
				extracted[key] = f
			} else {
				fmt.Printf("Error parsing %s: %v\n", key, err)
			}
		}
	}

	fmt.Printf("Extracted data: %v\n", extracted)
	c.JSON(http.StatusOK, gin.H{"extracted": extracted})
}
