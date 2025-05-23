package internal

import (
	"workflow-function/shared/schema"
)

// InputEvent represents the Lambda function input event
type InputEvent struct {
	VerificationContext schema.VerificationContext `json:"verificationContext"`
}

// OutputEvent represents the Lambda function output event
type OutputEvent struct {
	HistoricalContext HistoricalContext `json:"historicalContext"`
}

// HistoricalContext represents the historical verification data
type HistoricalContext struct {
	PreviousVerificationID        string              `json:"previousVerificationId"`
	PreviousVerificationAt        string              `json:"previousVerificationAt"`
	PreviousVerificationStatus    string              `json:"previousVerificationStatus"`
	HoursSinceLastVerification    float64             `json:"hoursSinceLastVerification"`
	MachineStructure              MachineStructure    `json:"machineStructure"`
	CheckingStatus                map[string]string   `json:"checkingStatus"`
	VerificationSummary           VerificationSummary `json:"verificationSummary"`
}

// MachineStructure represents the physical structure of the vending machine
type MachineStructure struct {
	RowCount      int      `json:"rowCount"`
	ColumnsPerRow int      `json:"columnsPerRow"`
	RowOrder      []string `json:"rowOrder"`
	ColumnOrder   []string `json:"columnOrder"`
}

// VerificationSummary contains the summary of the verification result
type VerificationSummary struct {
	TotalPositionsChecked  int     `json:"totalPositionsChecked"`
	CorrectPositions       int     `json:"correctPositions"`
	DiscrepantPositions    int     `json:"discrepantPositions"`
	MissingProducts        int     `json:"missingProducts"`
	IncorrectProductTypes  int     `json:"incorrectProductTypes"`
	UnexpectedProducts     int     `json:"unexpectedProducts"`
	EmptyPositionsCount    int     `json:"emptyPositionsCount"`
	OverallAccuracy        float64 `json:"overallAccuracy"`
	OverallConfidence      float64 `json:"overallConfidence"`
	VerificationStatus     string  `json:"verificationStatus"`
	VerificationOutcome    string  `json:"verificationOutcome"`
}

// VerificationRecord represents the DynamoDB record for verification results
type VerificationRecord struct {
	VerificationID         string              `json:"verificationId" dynamodbav:"VerificationId"`
	VerificationAt         string              `json:"verificationAt" dynamodbav:"VerificationAt"`
	VerificationType       string              `json:"verificationType" dynamodbav:"VerificationType"`
	VendingMachineID       string              `json:"vendingMachineId" dynamodbav:"VendingMachineId"`
	CheckingImageURL       string              `json:"checkingImageUrl" dynamodbav:"CheckingImageUrl"`
	ReferenceImageURL      string              `json:"referenceImageUrl" dynamodbav:"ReferenceImageUrl"`
	VerificationStatus     string              `json:"verificationStatus" dynamodbav:"VerificationStatus"`
	MachineStructure       MachineStructure    `json:"machineStructure" dynamodbav:"MachineStructure"`
	CheckingStatus         map[string]string   `json:"checkingStatus" dynamodbav:"CheckingStatus"`
	VerificationSummary    VerificationSummary `json:"verificationSummary" dynamodbav:"VerificationSummary"`
}