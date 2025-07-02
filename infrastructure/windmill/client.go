// Package windmill provides integration with Windmill workflow platform
package windmill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
)

// Client represents a Windmill API client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	workspace  string
}

// Config represents Windmill client configuration
type Config struct {
	BaseURL    string
	Token      string
	Workspace  string
	Timeout    time.Duration
}

// NewClient creates a new Windmill client
func NewClient(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL:   config.BaseURL,
		token:     config.Token,
		workspace: config.Workspace,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Workflow represents a Windmill workflow
type Workflow struct {
	Path        string                 `json:"path"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	Value       WorkflowValue          `json:"value"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
}

// WorkflowValue represents the workflow definition
type WorkflowValue struct {
	Modules   []WorkflowModule `json:"modules"`
	Failure   *FailureModule   `json:"failure,omitempty"`
	Same_worker bool           `json:"same_worker,omitempty"`
}

// WorkflowModule represents a module in a workflow
type WorkflowModule struct {
	ID          string                 `json:"id"`
	Value       ModuleValue            `json:"value"`
	StopAfterIf *string               `json:"stop_after_if,omitempty"`
	Sleep       *Sleep                `json:"sleep,omitempty"`
	Cache_ttl   *int                  `json:"cache_ttl,omitempty"`
	Timeout     *int                  `json:"timeout,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Mock        *Mock                 `json:"mock,omitempty"`
}

// ModuleValue represents the value of a workflow module
type ModuleValue struct {
	Type     string                 `json:"type"`
	Content  string                 `json:"content,omitempty"`
	Language string                 `json:"language,omitempty"`
	Path     string                 `json:"path,omitempty"`
	Input    map[string]interface{} `json:"input,omitempty"`
	Tag      string                 `json:"tag,omitempty"`
}

// FailureModule represents a failure handling module
type FailureModule struct {
	Modules []WorkflowModule `json:"modules"`
}

// Sleep represents a sleep configuration
type Sleep struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

// Mock represents a mock configuration
type Mock struct {
	Enabled bool        `json:"enabled"`
	Return  interface{} `json:"return"`
}

// Job represents a Windmill job
type Job struct {
	ID          string                 `json:"id"`
	WorkspaceID string                 `json:"workspace_id"`
	ParentJob   *string               `json:"parent_job"`
	CreatedBy   string                `json:"created_by"`
	CreatedAt   time.Time             `json:"created_at"`
	StartedAt   *time.Time            `json:"started_at"`
	CompletedAt *time.Time            `json:"completed_at"`
	JobKind     string                `json:"job_kind"`
	Script_hash *string               `json:"script_hash"`
	Script_path *string               `json:"script_path"`
	Args        map[string]interface{} `json:"args"`
	Result      interface{}           `json:"result"`
	Logs        string                `json:"logs"`
	Running     bool                  `json:"running"`
	Success     *bool                 `json:"success"`
	Error       *string               `json:"error"`
	Scheduled_for time.Time           `json:"scheduled_for"`
	Tag         string                `json:"tag"`
	Flow_status *FlowStatus           `json:"flow_status"`
	Raw_code    *string               `json:"raw_code"`
	Raw_lock    *string               `json:"raw_lock"`
	Language    string                `json:"language"`
	Is_skipped  bool                  `json:"is_skipped"`
	Is_flow_step bool                 `json:"is_flow_step"`
	Same_worker bool                  `json:"same_worker"`
}

// FlowStatus represents the status of a flow job
type FlowStatus struct {
	Step         int                    `json:"step"`
	Modules      []FlowStatusModule     `json:"modules"`
	Failure      *FlowStatusModule      `json:"failure"`
	RetryStatus  map[string]interface{} `json:"retry_status"`
}

// FlowStatusModule represents a module status in a flow
type FlowStatusModule struct {
	Type         string                 `json:"type"`
	Job          *string               `json:"job"`
	Count        *int                  `json:"count"`
	Iterator     *FlowIterator         `json:"iterator"`
	Flow_jobs    []string              `json:"flow_jobs"`
	Branch_chosen *BranchChosen        `json:"branch_chosen"`
}

// FlowIterator represents an iterator in a flow
type FlowIterator struct {
	Index int           `json:"index"`
	Itered interface{}  `json:"itered"`
	Args   []interface{} `json:"args"`
}

// BranchChosen represents a chosen branch
type BranchChosen struct {
	Type   string `json:"type"`
	Branch int    `json:"branch"`
}

// CreateWorkflowRequest represents a request to create a workflow
type CreateWorkflowRequest struct {
	Path        string                 `json:"path"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	Value       WorkflowValue          `json:"value"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
}

// RunWorkflowRequest represents a request to run a workflow
type RunWorkflowRequest struct {
	Args           map[string]interface{} `json:"args,omitempty"`
	ScheduledFor   *time.Time            `json:"scheduled_for,omitempty"`
	ScheduledInSecs *int                 `json:"scheduled_in_secs,omitempty"`
	ParentJob      *string               `json:"parent_job,omitempty"`
	Tag            *string               `json:"tag,omitempty"`
}

// CreateWorkflow creates a new workflow in Windmill
func (c *Client) CreateWorkflow(ctx context.Context, req CreateWorkflowRequest) result.Result[*Workflow] {
	url := fmt.Sprintf("%s/api/w/%s/flows", c.baseURL, c.workspace)
	
	jsonData, err := json.Marshal(req)
	if err != nil {
		return result.Failure[*Workflow](errors.InternalError("failed to marshal request: " + err.Error()))
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return result.Failure[*Workflow](errors.InternalError("failed to create request: " + err.Error()))
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return result.Failure[*Workflow](errors.InternalError("HTTP request failed: " + err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return result.Failure[*Workflow](errors.InternalError(fmt.Sprintf("API error: %d - %s", resp.StatusCode, string(body))))
	}

	var workflow Workflow
	if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
		return result.Failure[*Workflow](errors.InternalError("failed to decode response: " + err.Error()))
	}

	return result.Success(&workflow)
}

// RunWorkflow executes a workflow in Windmill
func (c *Client) RunWorkflow(ctx context.Context, path string, req RunWorkflowRequest) result.Result[*Job] {
	url := fmt.Sprintf("%s/api/w/%s/jobs/run/f/%s", c.baseURL, c.workspace, path)
	
	jsonData, err := json.Marshal(req)
	if err != nil {
		return result.Failure[*Job](errors.InternalError("failed to marshal request: " + err.Error()))
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return result.Failure[*Job](errors.InternalError("failed to create request: " + err.Error()))
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return result.Failure[*Job](errors.InternalError("HTTP request failed: " + err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return result.Failure[*Job](errors.InternalError(fmt.Sprintf("API error: %d - %s", resp.StatusCode, string(body))))
	}

	// Try to decode as job ID string first, then as object
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result.Failure[*Job](errors.InternalError("failed to read response: " + err.Error()))
	}

	var jobID string
	// Try direct string decode first
	if err := json.Unmarshal(body, &jobID); err != nil {
		// If that fails, try object decode
		var jobResponse struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(body, &jobResponse); err != nil {
			return result.Failure[*Job](errors.InternalError("failed to decode response: " + err.Error()))
		}
		jobID = jobResponse.ID
	}

	// Get the job details
	return c.GetJob(ctx, jobID)
}

// GetJob retrieves job information from Windmill
func (c *Client) GetJob(ctx context.Context, jobID string) result.Result[*Job] {
	url := fmt.Sprintf("%s/api/w/%s/jobs_u/get/%s", c.baseURL, c.workspace, jobID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result.Failure[*Job](errors.InternalError("failed to create request: " + err.Error()))
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return result.Failure[*Job](errors.InternalError("HTTP request failed: " + err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return result.Failure[*Job](errors.InternalError(fmt.Sprintf("API error: %d - %s", resp.StatusCode, string(body))))
	}

	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return result.Failure[*Job](errors.InternalError("failed to decode response: " + err.Error()))
	}

	return result.Success(&job)
}

// GetJobLogs retrieves job logs from Windmill
func (c *Client) GetJobLogs(ctx context.Context, jobID string) result.Result[string] {
	url := fmt.Sprintf("%s/api/w/%s/jobs_u/get_logs/%s", c.baseURL, c.workspace, jobID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result.Failure[string](errors.InternalError("failed to create request: " + err.Error()))
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return result.Failure[string](errors.InternalError("HTTP request failed: " + err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return result.Failure[string](errors.InternalError(fmt.Sprintf("API error: %d - %s", resp.StatusCode, string(body))))
	}

	logs, err := io.ReadAll(resp.Body)
	if err != nil {
		return result.Failure[string](errors.InternalError("failed to read logs: " + err.Error()))
	}

	return result.Success(string(logs))
}

// CancelJob cancels a running job
func (c *Client) CancelJob(ctx context.Context, jobID string, reason string) result.Result[bool] {
	url := fmt.Sprintf("%s/api/w/%s/jobs_u/cancel/%s", c.baseURL, c.workspace, jobID)

	reqBody := map[string]string{"reason": reason}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return result.Failure[bool](errors.InternalError("failed to marshal request: " + err.Error()))
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return result.Failure[bool](errors.InternalError("failed to create request: " + err.Error()))
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return result.Failure[bool](errors.InternalError("HTTP request failed: " + err.Error()))
	}
	defer resp.Body.Close()

	success := resp.StatusCode == http.StatusOK
	return result.Success(success)
}

// ListJobs lists jobs in the workspace
func (c *Client) ListJobs(ctx context.Context, createdBy, parentJob, scriptPathExact, scriptPathStart, scriptHash, startedBefore, startedAfter, success, running, scheduledForBefore, scheduledForAfter *string, jobKinds *string, limit *int, offset *int) result.Result[[]Job] {
	url := fmt.Sprintf("%s/api/w/%s/jobs/list", c.baseURL, c.workspace)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result.Failure[[]Job](errors.InternalError("failed to create request: " + err.Error()))
	}

	// Add query parameters
	q := httpReq.URL.Query()
	if createdBy != nil {
		q.Add("created_by", *createdBy)
	}
	if parentJob != nil {
		q.Add("parent_job", *parentJob)
	}
	if scriptPathExact != nil {
		q.Add("script_path_exact", *scriptPathExact)
	}
	if scriptPathStart != nil {
		q.Add("script_path_start", *scriptPathStart)
	}
	if scriptHash != nil {
		q.Add("script_hash", *scriptHash)
	}
	if startedBefore != nil {
		q.Add("started_before", *startedBefore)
	}
	if startedAfter != nil {
		q.Add("started_after", *startedAfter)
	}
	if success != nil {
		q.Add("success", *success)
	}
	if running != nil {
		q.Add("running", *running)
	}
	if scheduledForBefore != nil {
		q.Add("scheduled_for_before", *scheduledForBefore)
	}
	if scheduledForAfter != nil {
		q.Add("scheduled_for_after", *scheduledForAfter)
	}
	if jobKinds != nil {
		q.Add("job_kinds", *jobKinds)
	}
	if limit != nil {
		q.Add("per_page", fmt.Sprintf("%d", *limit))
	}
	if offset != nil {
		q.Add("page", fmt.Sprintf("%d", *offset))
	}
	httpReq.URL.RawQuery = q.Encode()

	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return result.Failure[[]Job](errors.InternalError("HTTP request failed: " + err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return result.Failure[[]Job](errors.InternalError(fmt.Sprintf("API error: %d - %s", resp.StatusCode, string(body))))
	}

	var jobs []Job
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return result.Failure[[]Job](errors.InternalError("failed to decode response: " + err.Error()))
	}

	return result.Success(jobs)
}

// GetWorkflow retrieves a workflow by path
func (c *Client) GetWorkflow(ctx context.Context, path string) result.Result[*Workflow] {
	url := fmt.Sprintf("%s/api/w/%s/flows/get/%s", c.baseURL, c.workspace, path)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result.Failure[*Workflow](errors.InternalError("failed to create request: " + err.Error()))
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return result.Failure[*Workflow](errors.InternalError("HTTP request failed: " + err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return result.Failure[*Workflow](errors.InternalError(fmt.Sprintf("API error: %d - %s", resp.StatusCode, string(body))))
	}

	var workflow Workflow
	if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
		return result.Failure[*Workflow](errors.InternalError("failed to decode response: " + err.Error()))
	}

	return result.Success(&workflow)
}

// DeleteWorkflow deletes a workflow
func (c *Client) DeleteWorkflow(ctx context.Context, path string) result.Result[bool] {
	url := fmt.Sprintf("%s/api/w/%s/flows/delete/%s", c.baseURL, c.workspace, path)

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return result.Failure[bool](errors.InternalError("failed to create request: " + err.Error()))
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return result.Failure[bool](errors.InternalError("HTTP request failed: " + err.Error()))
	}
	defer resp.Body.Close()

	success := resp.StatusCode == http.StatusOK
	return result.Success(success)
}

// WaitForJob waits for a job to complete with polling
func (c *Client) WaitForJob(ctx context.Context, jobID string, pollInterval time.Duration) result.Result[*Job] {
	if pollInterval == 0 {
		pollInterval = 2 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return result.Failure[*Job](errors.InternalError("context canceled while waiting for job"))
		case <-ticker.C:
			jobResult := c.GetJob(ctx, jobID)
			if jobResult.IsFailure() {
				return jobResult
			}

			job := jobResult.Value()
			if !job.Running {
				return result.Success(job)
			}
		}
	}
}