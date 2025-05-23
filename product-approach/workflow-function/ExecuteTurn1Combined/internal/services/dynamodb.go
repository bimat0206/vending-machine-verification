// internal/services/dynamodb.go - ENHANCED VERSION WITH COMPREHENSIVE DESIGN INTEGRATION
package services

import (
	"context"
	goerrors "errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"workflow-function/ExecuteTurn1Combined/internal/config"
	"workflow-function/ExecuteTurn1Combined/internal/models"

	// Using shared packages for enhanced functionality
	"workflow-function/shared/errors"
	"workflow-function/shared/schema"
)

// DynamoDBService defines enhanced status-tracking, conversation-history, and metrics operations.
type DynamoDBService interface {
	// Legacy methods (maintained for backward compatibility)
	UpdateVerificationStatus(ctx context.Context, verificationID string, status models.VerificationStatus, metrics models.TokenUsage) error
	RecordConversationTurn(ctx context.Context, turn *models.ConversationTurn) error

	// Enhanced methods for comprehensive design integration
	UpdateVerificationStatusEnhanced(ctx context.Context, verificationID string, statusEntry schema.StatusHistoryEntry) error
	RecordConversationHistory(ctx context.Context, conversationTracker *schema.ConversationTracker) error
	UpdateProcessingMetrics(ctx context.Context, verificationID string, metrics *schema.ProcessingMetrics) error
	UpdateStatusHistory(ctx context.Context, verificationID string, statusHistory []schema.StatusHistoryEntry) error
	UpdateErrorTracking(ctx context.Context, verificationID string, errorTracking *schema.ErrorTracking) error

	// Real-time status tracking methods
	InitializeVerificationRecord(ctx context.Context, verificationContext *schema.VerificationContext) error
	UpdateCurrentStatus(ctx context.Context, verificationID, currentStatus, lastUpdatedAt string, metrics map[string]interface{}) error
	GetVerificationStatus(ctx context.Context, verificationID string) (*VerificationStatusInfo, error)

	// Conversation management methods
	InitializeConversationHistory(ctx context.Context, verificationID string, maxTurns int, metadata map[string]interface{}) error
	UpdateConversationTurn(ctx context.Context, verificationID string, turnData *schema.TurnResponse) error
	CompleteConversation(ctx context.Context, verificationID string, finalStatus string) error

	// Query methods for historical data (supporting Use Case 2)
	QueryPreviousVerification(ctx context.Context, checkingImageUrl string) (*schema.VerificationContext, error)
	GetLayoutMetadata(ctx context.Context, layoutID int, layoutPrefix string) (*schema.LayoutMetadata, error)
}

// VerificationStatusInfo represents current verification status information
type VerificationStatusInfo struct {
	VerificationID    string                      `json:"verificationId"`
	CurrentStatus     string                      `json:"currentStatus"`
	LastUpdatedAt     string                      `json:"lastUpdatedAt"`
	StatusHistory     []schema.StatusHistoryEntry `json:"statusHistory"`
	ProcessingMetrics *schema.ProcessingMetrics   `json:"processingMetrics"`
	ErrorTracking     *schema.ErrorTracking       `json:"errorTracking"`
}

type dynamoClient struct {
	client            *dynamodb.Client
	verificationTable string
	conversationTable string
	layoutTable       string
	region            string
}

// NewDynamoDBService constructs an enhanced DynamoDBService with comprehensive capabilities.
func NewDynamoDBService(cfg *config.Config) (DynamoDBService, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(cfg.AWS.Region))
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeInternal,
			"failed to load AWS config for DynamoDB", false).
			WithContext("region", cfg.AWS.Region)
	}
	client := dynamodb.NewFromConfig(awsCfg)
	return &dynamoClient{
		client:            client,
		verificationTable: cfg.AWS.DynamoDBVerificationTable,
		conversationTable: cfg.AWS.DynamoDBConversationTable,
		layoutTable:       "LayoutMetadata", // Would be configurable in real implementation
		region:            cfg.AWS.Region,
	}, nil
}

// Legacy method: UpdateVerificationStatus sets the verification's currentStatus and tokenUsage.
func (d *dynamoClient) UpdateVerificationStatus(ctx context.Context, verificationID string, status models.VerificationStatus, metrics models.TokenUsage) error {
	// Marshal metrics struct into DynamoDB attribute map
	avMetrics, err := attributevalue.MarshalMap(metrics)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to marshal token usage metrics", true)
	}

	input := &dynamodb.UpdateItemInput{
		TableName: &d.verificationTable,
		Key: map[string]types.AttributeValue{
			"verificationId": &types.AttributeValueMemberS{Value: verificationID},
		},
		UpdateExpression: aws.String("SET currentStatus = :status, tokenUsage = :metrics, lastUpdatedAt = :updated"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":  &types.AttributeValueMemberS{Value: string(status)},
			":metrics": &types.AttributeValueMemberM{Value: avMetrics},
			":updated": &types.AttributeValueMemberS{Value: schema.FormatISO8601()},
		},
	}

	if _, err := d.client.UpdateItem(ctx, input); err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to update verification status", true).
			WithContext("table", d.verificationTable).
			WithContext("verificationId", verificationID).
			WithContext("operation", "UpdateItem")
	}
	return nil
}

// Enhanced method: UpdateVerificationStatusEnhanced updates status with comprehensive tracking.
func (d *dynamoClient) UpdateVerificationStatusEnhanced(ctx context.Context, verificationID string, statusEntry schema.StatusHistoryEntry) error {
	// Marshal status entry
	avStatusEntry, err := attributevalue.MarshalMap(statusEntry)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to marshal status history entry", true).
			WithContext("verificationId", verificationID).
			WithContext("status", statusEntry.Status)
	}

	// Update expression to append to status history and update current status
	input := &dynamodb.UpdateItemInput{
		TableName: &d.verificationTable,
		Key: map[string]types.AttributeValue{
			"verificationId": &types.AttributeValueMemberS{Value: verificationID},
		},
		UpdateExpression: aws.String("SET currentStatus = :status, lastUpdatedAt = :updated, " +
			"statusHistory = list_append(if_not_exists(statusHistory, :empty), :entry)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":  &types.AttributeValueMemberS{Value: statusEntry.Status},
			":updated": &types.AttributeValueMemberS{Value: statusEntry.Timestamp},
			":entry":   &types.AttributeValueMemberL{Value: []types.AttributeValue{&types.AttributeValueMemberM{Value: avStatusEntry}}},
			":empty":   &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		},
	}

	if _, err := d.client.UpdateItem(ctx, input); err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to update verification status with history", true).
			WithContext("table", d.verificationTable).
			WithContext("verificationId", verificationID).
			WithContext("status", statusEntry.Status).
			WithContext("operation", "UpdateItemEnhanced")
	}
	return nil
}

// Legacy method: RecordConversationTurn inserts a new item into the conversation history table.
func (d *dynamoClient) RecordConversationTurn(ctx context.Context, turn *models.ConversationTurn) error {
	item, err := attributevalue.MarshalMap(turn)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to marshal conversation turn", true).
			WithContext("turn_id", turn.TurnID).
			WithContext("verification_id", turn.VerificationID)
	}

	input := &dynamodb.PutItemInput{
		TableName: &d.conversationTable,
		Item:      item,
	}

	if _, err := d.client.PutItem(ctx, input); err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to record conversation turn", true).
			WithContext("table", d.conversationTable).
			WithContext("turn_id", turn.TurnID).
			WithContext("verification_id", turn.VerificationID).
			WithContext("operation", "PutItem")
	}
	return nil
}

// Enhanced method: RecordConversationHistory manages comprehensive conversation tracking.
func (d *dynamoClient) RecordConversationHistory(ctx context.Context, conversationTracker *schema.ConversationTracker) error {
	item, err := attributevalue.MarshalMap(conversationTracker)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to marshal conversation tracker", true).
			WithContext("conversation_id", conversationTracker.ConversationId).
			WithContext("current_turn", conversationTracker.CurrentTurn)
	}

	// Use PutItem with condition to prevent overwrites
	input := &dynamodb.PutItemInput{
		TableName:           &d.conversationTable,
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(conversationId) OR conversationAt < :newTime"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":newTime": &types.AttributeValueMemberS{Value: conversationTracker.ConversationAt},
		},
	}

	if _, err := d.client.PutItem(ctx, input); err != nil {
		// Handle conditional check failure gracefully
		var conditionalCheckErr *types.ConditionalCheckFailedException
		if goerrors.As(err, &conditionalCheckErr) {
			// Update existing record instead
			return d.updateExistingConversationHistory(ctx, conversationTracker)
		}

		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to record conversation history", true).
			WithContext("table", d.conversationTable).
			WithContext("conversation_id", conversationTracker.ConversationId).
			WithContext("operation", "PutItemWithCondition")
	}
	return nil
}

// Helper method to update existing conversation history
func (d *dynamoClient) updateExistingConversationHistory(ctx context.Context, conversationTracker *schema.ConversationTracker) error {
	// Marshal history and metadata
	avHistory, err := attributevalue.MarshalList(conversationTracker.History)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to marshal conversation history", true)
	}

	avMetadata, err := attributevalue.MarshalMap(conversationTracker.Metadata)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to marshal conversation metadata", true)
	}

	input := &dynamodb.UpdateItemInput{
		TableName: &d.conversationTable,
		Key: map[string]types.AttributeValue{
			"conversationId": &types.AttributeValueMemberS{Value: conversationTracker.ConversationId},
			"conversationAt": &types.AttributeValueMemberS{Value: conversationTracker.ConversationAt},
		},
		UpdateExpression: aws.String("SET currentTurn = :turn, turnStatus = :status, history = :history, metadata = :metadata"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":turn":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", conversationTracker.CurrentTurn)},
			":status":   &types.AttributeValueMemberS{Value: conversationTracker.TurnStatus},
			":history":  &types.AttributeValueMemberL{Value: avHistory},
			":metadata": &types.AttributeValueMemberM{Value: avMetadata},
		},
	}

	if _, err := d.client.UpdateItem(ctx, input); err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to update conversation history", true).
			WithContext("conversation_id", conversationTracker.ConversationId)
	}
	return nil
}

// Enhanced method: UpdateProcessingMetrics stores comprehensive performance metrics.
func (d *dynamoClient) UpdateProcessingMetrics(ctx context.Context, verificationID string, metrics *schema.ProcessingMetrics) error {
	avMetrics, err := attributevalue.MarshalMap(metrics)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to marshal processing metrics", true).
			WithContext("verificationId", verificationID)
	}

	input := &dynamodb.UpdateItemInput{
		TableName: &d.verificationTable,
		Key: map[string]types.AttributeValue{
			"verificationId": &types.AttributeValueMemberS{Value: verificationID},
		},
		UpdateExpression: aws.String("SET processingMetrics = :metrics, lastUpdatedAt = :updated"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":metrics": &types.AttributeValueMemberM{Value: avMetrics},
			":updated": &types.AttributeValueMemberS{Value: schema.FormatISO8601()},
		},
	}

	if _, err := d.client.UpdateItem(ctx, input); err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to update processing metrics", true).
			WithContext("table", d.verificationTable).
			WithContext("verificationId", verificationID).
			WithContext("operation", "UpdateProcessingMetrics")
	}
	return nil
}

// Enhanced method: UpdateStatusHistory maintains complete status progression tracking.
func (d *dynamoClient) UpdateStatusHistory(ctx context.Context, verificationID string, statusHistory []schema.StatusHistoryEntry) error {
	avStatusHistory, err := attributevalue.MarshalList(statusHistory)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to marshal status history", true).
			WithContext("verificationId", verificationID).
			WithContext("history_count", len(statusHistory))
	}

	input := &dynamodb.UpdateItemInput{
		TableName: &d.verificationTable,
		Key: map[string]types.AttributeValue{
			"verificationId": &types.AttributeValueMemberS{Value: verificationID},
		},
		UpdateExpression: aws.String("SET statusHistory = :history, lastUpdatedAt = :updated"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":history": &types.AttributeValueMemberL{Value: avStatusHistory},
			":updated": &types.AttributeValueMemberS{Value: schema.FormatISO8601()},
		},
	}

	if _, err := d.client.UpdateItem(ctx, input); err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to update status history", true).
			WithContext("table", d.verificationTable).
			WithContext("verificationId", verificationID).
			WithContext("history_count", len(statusHistory))
	}
	return nil
}

// Enhanced method: UpdateErrorTracking manages comprehensive error state tracking.
func (d *dynamoClient) UpdateErrorTracking(ctx context.Context, verificationID string, errorTracking *schema.ErrorTracking) error {
	avErrorTracking, err := attributevalue.MarshalMap(errorTracking)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to marshal error tracking", true).
			WithContext("verificationId", verificationID)
	}

	input := &dynamodb.UpdateItemInput{
		TableName: &d.verificationTable,
		Key: map[string]types.AttributeValue{
			"verificationId": &types.AttributeValueMemberS{Value: verificationID},
		},
		UpdateExpression: aws.String("SET errorTracking = :tracking, lastUpdatedAt = :updated"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":tracking": &types.AttributeValueMemberM{Value: avErrorTracking},
			":updated":  &types.AttributeValueMemberS{Value: schema.FormatISO8601()},
		},
	}

	if _, err := d.client.UpdateItem(ctx, input); err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to update error tracking", true).
			WithContext("table", d.verificationTable).
			WithContext("verificationId", verificationID).
			WithContext("has_errors", errorTracking.HasErrors)
	}
	return nil
}

// Real-time tracking: InitializeVerificationRecord creates the initial verification record.
func (d *dynamoClient) InitializeVerificationRecord(ctx context.Context, verificationContext *schema.VerificationContext) error {
	item, err := attributevalue.MarshalMap(verificationContext)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to marshal verification context", true).
			WithContext("verificationId", verificationContext.VerificationId)
	}

	input := &dynamodb.PutItemInput{
		TableName:           &d.verificationTable,
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(verificationId)"),
	}

	if _, err := d.client.PutItem(ctx, input); err != nil {
		var conditionalCheckErr *types.ConditionalCheckFailedException
		if goerrors.As(err, &conditionalCheckErr) {
			return errors.NewValidationError("verification record already exists", map[string]interface{}{
				"verificationId": verificationContext.VerificationId,
			})
		}

		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to initialize verification record", true).
			WithContext("table", d.verificationTable).
			WithContext("verificationId", verificationContext.VerificationId)
	}
	return nil
}

// Real-time tracking: UpdateCurrentStatus updates the current processing status.
func (d *dynamoClient) UpdateCurrentStatus(ctx context.Context, verificationID, currentStatus, lastUpdatedAt string, metrics map[string]interface{}) error {
	updateExpression := "SET currentStatus = :status, lastUpdatedAt = :updated"
	expressionValues := map[string]types.AttributeValue{
		":status":  &types.AttributeValueMemberS{Value: currentStatus},
		":updated": &types.AttributeValueMemberS{Value: lastUpdatedAt},
	}

	// Add metrics if provided
	if metrics != nil && len(metrics) > 0 {
		avMetrics, err := attributevalue.MarshalMap(metrics)
		if err != nil {
			return errors.WrapError(err, errors.ErrorTypeDynamoDB,
				"failed to marshal status metrics", true)
		}
		updateExpression += ", statusMetrics = :metrics"
		expressionValues[":metrics"] = &types.AttributeValueMemberM{Value: avMetrics}
	}

	input := &dynamodb.UpdateItemInput{
		TableName: &d.verificationTable,
		Key: map[string]types.AttributeValue{
			"verificationId": &types.AttributeValueMemberS{Value: verificationID},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionValues,
	}

	if _, err := d.client.UpdateItem(ctx, input); err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to update current status", true).
			WithContext("verificationId", verificationID).
			WithContext("status", currentStatus)
	}
	return nil
}

// Real-time tracking: GetVerificationStatus retrieves current verification status.
func (d *dynamoClient) GetVerificationStatus(ctx context.Context, verificationID string) (*VerificationStatusInfo, error) {
	input := &dynamodb.GetItemInput{
		TableName: &d.verificationTable,
		Key: map[string]types.AttributeValue{
			"verificationId": &types.AttributeValueMemberS{Value: verificationID},
		},
		ProjectionExpression: aws.String("verificationId, currentStatus, lastUpdatedAt, statusHistory, processingMetrics, errorTracking"),
	}

	result, err := d.client.GetItem(ctx, input)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to get verification status", true).
			WithContext("verificationId", verificationID)
	}

	if result.Item == nil {
		return nil, errors.NewValidationError("verification not found", map[string]interface{}{
			"verificationId": verificationID,
		})
	}

	var statusInfo VerificationStatusInfo
	if err := attributevalue.UnmarshalMap(result.Item, &statusInfo); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to unmarshal verification status", false).
			WithContext("verificationId", verificationID)
	}

	return &statusInfo, nil
}

// Conversation management: InitializeConversationHistory creates initial conversation record.
func (d *dynamoClient) InitializeConversationHistory(ctx context.Context, verificationID string, maxTurns int, metadata map[string]interface{}) error {
	conversationTracker := &schema.ConversationTracker{
		ConversationId: verificationID,
		CurrentTurn:    0,
		MaxTurns:       maxTurns,
		TurnStatus:     "INITIALIZED",
		ConversationAt: schema.FormatISO8601(),
		History:        make([]interface{}, 0),
		Metadata:       metadata,
	}

	return d.RecordConversationHistory(ctx, conversationTracker)
}

// Conversation management: UpdateConversationTurn adds a new turn to the conversation.
func (d *dynamoClient) UpdateConversationTurn(ctx context.Context, verificationID string, turnData *schema.TurnResponse) error {
	// Query to find the most recent conversation record for this verificationID
	queryInput := &dynamodb.QueryInput{
		TableName:              &d.conversationTable,
		KeyConditionExpression: aws.String("conversationId = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberS{Value: verificationID},
		},
		ScanIndexForward: aws.Bool(false), // Get most recent first
		Limit:            aws.Int32(1),
	}

	queryResult, err := d.client.Query(ctx, queryInput)
	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to query conversation for update", true).
			WithContext("verificationId", verificationID)
	}

	var conversationTracker schema.ConversationTracker
	if len(queryResult.Items) > 0 {
		if err := attributevalue.UnmarshalMap(queryResult.Items[0], &conversationTracker); err != nil {
			return errors.WrapError(err, errors.ErrorTypeDynamoDB,
				"failed to unmarshal conversation tracker", false).
				WithContext("verificationId", verificationID)
		}
	} else {
		// Initialize if not exists
		conversationTracker = schema.ConversationTracker{
			ConversationId: verificationID,
			CurrentTurn:    0,
			MaxTurns:       2,
			TurnStatus:     "ACTIVE",
			ConversationAt: schema.FormatISO8601(),
			History:        make([]interface{}, 0),
			Metadata:       make(map[string]interface{}),
		}
	}

	// Add new turn to history
	turnEntry := map[string]interface{}{
		"turnId":     turnData.TurnId,
		"timestamp":  turnData.Timestamp,
		"stage":      turnData.Stage,
		"latencyMs":  turnData.LatencyMs,
		"tokenUsage": turnData.TokenUsage,
		"s3References": map[string]interface{}{
			"prompt":   fmt.Sprintf("prompts/turn%d-prompt.json", turnData.TurnId),
			"response": fmt.Sprintf("responses/turn%d-response.json", turnData.TurnId),
		},
	}

	conversationTracker.History = append(conversationTracker.History, turnEntry)
	conversationTracker.CurrentTurn = turnData.TurnId
	// Keep the existing ConversationAt timestamp if updating, only set new one if creating
	if len(queryResult.Items) == 0 {
		// This is a new conversation, so we set the timestamp
		conversationTracker.ConversationAt = schema.FormatISO8601()
	}

	// Use updateExistingConversationHistory directly if record exists
	if len(queryResult.Items) > 0 {
		return d.updateExistingConversationHistory(ctx, &conversationTracker)
	}
	return d.RecordConversationHistory(ctx, &conversationTracker)
}

// Conversation management: CompleteConversation marks conversation as completed.
func (d *dynamoClient) CompleteConversation(ctx context.Context, verificationID string, finalStatus string) error {
	input := &dynamodb.UpdateItemInput{
		TableName: &d.conversationTable,
		Key: map[string]types.AttributeValue{
			"conversationId": &types.AttributeValueMemberS{Value: verificationID},
		},
		UpdateExpression: aws.String("SET turnStatus = :status, conversationAt = :updated"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":  &types.AttributeValueMemberS{Value: finalStatus},
			":updated": &types.AttributeValueMemberS{Value: schema.FormatISO8601()},
		},
	}

	if _, err := d.client.UpdateItem(ctx, input); err != nil {
		return errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to complete conversation", true).
			WithContext("verificationId", verificationID).
			WithContext("finalStatus", finalStatus)
	}
	return nil
}

// Query methods: QueryPreviousVerification supports Use Case 2 historical lookup.
func (d *dynamoClient) QueryPreviousVerification(ctx context.Context, checkingImageUrl string) (*schema.VerificationContext, error) {
	// Query GSI4 (checkingImageUrl-verificationAt) to find most recent verification
	input := &dynamodb.QueryInput{
		TableName:              &d.verificationTable,
		IndexName:              aws.String("GSI4"),
		KeyConditionExpression: aws.String("checkingImageUrl = :imageUrl"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":imageUrl": &types.AttributeValueMemberS{Value: checkingImageUrl},
		},
		ScanIndexForward: aws.Bool(false), // Most recent first
		Limit:            aws.Int32(1),    // Only need the most recent
	}

	result, err := d.client.Query(ctx, input)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to query previous verification", true).
			WithContext("checkingImageUrl", checkingImageUrl).
			WithContext("index", "GSI4")
	}

	if len(result.Items) == 0 {
		return nil, errors.NewValidationError("no previous verification found", map[string]interface{}{
			"checkingImageUrl": checkingImageUrl,
		})
	}

	var verificationContext schema.VerificationContext
	if err := attributevalue.UnmarshalMap(result.Items[0], &verificationContext); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to unmarshal previous verification", false).
			WithContext("checkingImageUrl", checkingImageUrl)
	}

	return &verificationContext, nil
}

// Query methods: GetLayoutMetadata retrieves layout information for Use Case 1.
func (d *dynamoClient) GetLayoutMetadata(ctx context.Context, layoutID int, layoutPrefix string) (*schema.LayoutMetadata, error) {
	input := &dynamodb.GetItemInput{
		TableName: &d.layoutTable,
		Key: map[string]types.AttributeValue{
			"layoutId":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", layoutID)},
			"layoutPrefix": &types.AttributeValueMemberS{Value: layoutPrefix},
		},
	}

	result, err := d.client.GetItem(ctx, input)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to get layout metadata", true).
			WithContext("layoutId", layoutID).
			WithContext("layoutPrefix", layoutPrefix)
	}

	if result.Item == nil {
		return nil, errors.NewValidationError("layout metadata not found", map[string]interface{}{
			"layoutId":     layoutID,
			"layoutPrefix": layoutPrefix,
		})
	}

	var layoutMetadata schema.LayoutMetadata
	if err := attributevalue.UnmarshalMap(result.Item, &layoutMetadata); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeDynamoDB,
			"failed to unmarshal layout metadata", false).
			WithContext("layoutId", layoutID).
			WithContext("layoutPrefix", layoutPrefix)
	}

	return &layoutMetadata, nil
}

// Helper function to create AWS string pointer
func awsString(s string) *string {
	return &s
}

// Helper function to create AWS int32 pointer
func awsInt32(i int32) *int32 {
	return &i
}

// Helper function to create AWS bool pointer
func awsBool(b bool) *bool {
	return &b
}

// Additional helper methods for batch operations (for future enhancement)
func (d *dynamoClient) BatchUpdateVerificationStatuses(ctx context.Context, updates []VerificationStatusUpdate) error {
	// Implementation for batch updates - placeholder for future enhancement
	// This would be useful for updating multiple verifications simultaneously
	return nil
}

// Additional helper methods for transaction operations (for future enhancement)
func (d *dynamoClient) TransactUpdateVerificationAndConversation(ctx context.Context, verificationUpdate interface{}, conversationUpdate interface{}) error {
	// Implementation for transactional updates - placeholder for future enhancement
	// This would ensure atomicity between verification and conversation updates
	return nil
}

// VerificationStatusUpdate represents a batch status update
type VerificationStatusUpdate struct {
	VerificationID string                    `json:"verificationId"`
	StatusEntry    schema.StatusHistoryEntry `json:"statusEntry"`
	Metrics        *schema.ProcessingMetrics `json:"metrics,omitempty"`
}
