/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package pipelineConfig

import (
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"time"
)

type CiWorkflowRepository interface {
	SaveWorkFlowConfig(config *CiWorkflowConfig) error
	FindConfigByPipelineId(pipelineId int) (*CiWorkflowConfig, error)

	SaveWorkFlow(wf *CiWorkflow) error
	FindLastTriggeredWorkflow(pipelineId int) (*CiWorkflow, error)
	UpdateWorkFlow(wf *CiWorkflow) error
	FindByStatusesIn(activeStatuses []string) ([]*CiWorkflow, error)
	FindByPipelineId(pipelineId int, offset int, size int) ([]WorkflowWithArtifact, error)
	FindById(id int) (*CiWorkflow, error)
	FindCiWorkflowGitTriggersById(id int) (workflow *CiWorkflow, err error)
	FindByName(name string) (*CiWorkflow, error)

	FindLastTriggeredWorkflowByCiIds(pipelineId []int) (ciWorkflow []*CiWorkflow, err error)
	FindLastTriggeredWorkflowByArtifactId(ciArtifactId int) (ciWorkflow *CiWorkflow, err error)
	FindAllLastTriggeredWorkflowByArtifactId(ciArtifactId []int) (ciWorkflow []*CiWorkflow, err error)
	FindLastTriggeredWorkflowGitTriggersByArtifactId(ciArtifactId int) (ciWorkflow *CiWorkflow, err error)
	ExistsByStatus(status string) (bool, error)
	FindBuildTypeAndStatusDataOfLast1Day() []*BuildTypeCount
	FIndCiWorkflowStatusesByAppId(appId int) ([]*CiWorkflowStatus, error)
}

type CiWorkflowRepositoryImpl struct {
	dbConnection *pg.DB
	logger       *zap.SugaredLogger
}

type CiWorkflow struct {
	tableName          struct{}          `sql:"ci_workflow" pg:",discard_unknown_columns"`
	Id                 int               `sql:"id,pk"`
	Name               string            `sql:"name"`
	Status             string            `sql:"status"`
	PodStatus          string            `sql:"pod_status"`
	Message            string            `sql:"message"`
	StartedOn          time.Time         `sql:"started_on"`
	FinishedOn         time.Time         `sql:"finished_on"`
	CiPipelineId       int               `sql:"ci_pipeline_id"`
	Namespace          string            `sql:"namespace"`
	BlobStorageEnabled bool              `sql:"blob_storage_enabled,notnull"`
	LogLocation        string            `sql:"log_file_path"`
	GitTriggers        map[int]GitCommit `sql:"git_triggers"`
	TriggeredBy        int32             `sql:"triggered_by"`
	CiArtifactLocation string            `sql:"ci_artifact_location"`
	PodName            string            `sql:"pod_name"`
	CiBuildType        string            `sql:"ci_build_type"`
	EnvironmentId      int               `sql:"environment_id"`
	CiPipeline         *CiPipeline
}

type WorkflowWithArtifact struct {
	Id                 int               `json:"id"`
	Name               string            `json:"name"`
	PodName            string            `json:"podName"`
	Status             string            `json:"status"`
	PodStatus          string            `json:"pod_status"`
	Message            string            `json:"message"`
	StartedOn          time.Time         `json:"started_on"`
	FinishedOn         time.Time         `json:"finished_on"`
	CiPipelineId       int               `json:"ci_pipeline_id"`
	Namespace          string            `json:"namespace"`
	LogFilePath        string            `json:"log_file_path"`
	GitTriggers        map[int]GitCommit `json:"git_triggers"`
	TriggeredBy        int32             `json:"triggered_by"`
	EmailId            string            `json:"email_id"`
	Image              string            `json:"image"`
	CiArtifactLocation string            `json:"ci_artifact_location"`
	CiArtifactId       int               `json:"ci_artifact_d"`
	BlobStorageEnabled bool              `json:"blobStorageEnabled"`
	CiBuildType        string            `json:"ci_build_type"`
	IsArtifactUploaded bool              `json:"is_artifact_uploaded"`
	EnvironmentId      int               `json:"environmentId"`
	EnvironmentName    string            `json:"environmentName"`
}

type GitCommit struct {
	Commit                 string //git hash
	Author                 string
	Date                   time.Time
	Message                string
	Changes                []string
	WebhookData            WebhookData
	CiConfigureSourceValue string
	GitRepoUrl             string
	GitRepoName            string
	CiConfigureSourceType  SourceType
}

type WebhookData struct {
	Id              int
	EventActionType string
	Data            map[string]string
}

type CiWorkflowConfig struct {
	tableName                struct{} `sql:"ci_workflow_config" pg:",discard_unknown_columns"`
	Id                       int      `sql:"id,pk"`
	CiTimeout                int64    `sql:"ci_timeout"`
	MinCpu                   string   `sql:"min_cpu"`
	MaxCpu                   string   `sql:"max_cpu"`
	MinMem                   string   `sql:"min_mem"`
	MaxMem                   string   `sql:"max_mem"`
	MinStorage               string   `sql:"min_storage"`
	MaxStorage               string   `sql:"max_storage"`
	MinEphStorage            string   `sql:"min_eph_storage"`
	MaxEphStorage            string   `sql:"max_eph_storage"`
	CiCacheBucket            string   `sql:"ci_cache_bucket"`
	CiCacheRegion            string   `sql:"ci_cache_region"`
	CiImage                  string   `sql:"ci_image"`
	Namespace                string   `sql:"wf_namespace"`
	CiPipelineId             int      `sql:"ci_pipeline_id"`
	LogsBucket               string   `sql:"logs_bucket"`
	CiArtifactLocationFormat string   `sql:"ci_artifact_location_format"`
}

func NewCiWorkflowRepositoryImpl(dbConnection *pg.DB, logger *zap.SugaredLogger) *CiWorkflowRepositoryImpl {
	return &CiWorkflowRepositoryImpl{
		dbConnection: dbConnection,
		logger:       logger,
	}
}

func (impl *CiWorkflowRepositoryImpl) FindLastTriggeredWorkflow(pipelineId int) (ciWorkflow *CiWorkflow, err error) {
	workflow := &CiWorkflow{}
	err = impl.dbConnection.Model(workflow).
		Column("ci_workflow.*", "CiPipeline").
		Where("ci_workflow.ci_pipeline_id = ? ", pipelineId).
		Order("ci_workflow.started_on Desc").
		Limit(1).
		Select()
	return workflow, err
}

func (impl *CiWorkflowRepositoryImpl) FindByStatusesIn(activeStatuses []string) ([]*CiWorkflow, error) {
	var ciWorkFlows []*CiWorkflow
	err := impl.dbConnection.Model(&ciWorkFlows).
		Column("ci_workflow.*").
		Where("ci_workflow.status in (?)", pg.In(activeStatuses)).
		Select()
	return ciWorkFlows, err
}

func (impl *CiWorkflowRepositoryImpl) FindByPipelineId(pipelineId int, offset int, limit int) ([]WorkflowWithArtifact, error) {
	var wfs []WorkflowWithArtifact
	queryTemp := "select cia.id as ci_artifact_id, env.environment_name, cia.image, cia.is_artifact_uploaded, wf.*, u.email_id from ci_workflow wf left join users u on u.id = wf.triggered_by left join ci_artifact cia on wf.id = cia.ci_workflow_id left join environment env on env.id = wf.environment_id where wf.ci_pipeline_id = ? order by wf.started_on desc offset ? limit ?;"
	_, err := impl.dbConnection.Query(&wfs, queryTemp, pipelineId, offset, limit)
	if err != nil {
		return nil, err
	}
	return wfs, err
}

func (impl *CiWorkflowRepositoryImpl) FindByName(name string) (*CiWorkflow, error) {
	var ciWorkFlow *CiWorkflow
	err := impl.dbConnection.Model(&ciWorkFlow).
		Column("ci_workflow.*").
		Where("ci_workflow.name = ?", name).
		Select()
	return ciWorkFlow, err
}

func (impl *CiWorkflowRepositoryImpl) FindById(id int) (*CiWorkflow, error) {
	workflow := &CiWorkflow{}
	err := impl.dbConnection.Model(workflow).
		Column("ci_workflow.*", "CiPipeline", "CiPipeline.App", "CiPipeline.CiTemplate", "CiPipeline.CiTemplate.DockerRegistry").
		Where("ci_workflow.id = ? ", id).
		Select()
	return workflow, err
}

func (impl *CiWorkflowRepositoryImpl) FindCiWorkflowGitTriggersById(id int) (ciWorkflow *CiWorkflow, err error) {
	workflow := &CiWorkflow{}
	err = impl.dbConnection.Model(workflow).
		Column("ci_workflow.git_triggers").
		Where("ci_workflow.id = ? ", id).
		Select()

	return workflow, err
}

func (impl *CiWorkflowRepositoryImpl) SaveWorkFlowConfig(config *CiWorkflowConfig) error {
	err := impl.dbConnection.Insert(config)
	return err
}

func (impl *CiWorkflowRepositoryImpl) FindConfigByPipelineId(pipelineId int) (*CiWorkflowConfig, error) {
	ciWorkflowConfig := &CiWorkflowConfig{}
	err := impl.dbConnection.Model(ciWorkflowConfig).Where("ci_pipeline_id = ?", pipelineId).Select()
	return ciWorkflowConfig, err
}

func (impl *CiWorkflowRepositoryImpl) SaveWorkFlow(wf *CiWorkflow) error {
	err := impl.dbConnection.Insert(wf)
	return err
}

func (impl *CiWorkflowRepositoryImpl) UpdateWorkFlow(wf *CiWorkflow) error {
	err := impl.dbConnection.Update(wf)
	return err
}

func (impl *CiWorkflowRepositoryImpl) FindLastTriggeredWorkflowByCiIds(pipelineId []int) (ciWorkflow []*CiWorkflow, err error) {
	err = impl.dbConnection.Model(&ciWorkflow).
		Column("ci_workflow.*", "CiPipeline").
		Where("ci_workflow.ci_pipeline_id in (?) ", pg.In(pipelineId)).
		Order("ci_workflow.started_on Desc").
		Select()
	return ciWorkflow, err
}

func (impl *CiWorkflowRepositoryImpl) FindLastTriggeredWorkflowByArtifactId(ciArtifactId int) (ciWorkflow *CiWorkflow, err error) {
	workflow := &CiWorkflow{}
	err = impl.dbConnection.Model(workflow).
		Column("ci_workflow.*", "CiPipeline").
		Join("inner join ci_artifact cia on cia.ci_workflow_id = ci_workflow.id").
		Where("cia.id = ? ", ciArtifactId).
		Select()
	return workflow, err
}

func (impl *CiWorkflowRepositoryImpl) FindAllLastTriggeredWorkflowByArtifactId(ciArtifactIds []int) (ciWorkflows []*CiWorkflow, err error) {
	err = impl.dbConnection.Model(&ciWorkflows).
		Column("ci_workflow.git_triggers", "ci_workflow.ci_pipeline_id", "CiPipeline", "cia.id").
		Join("inner join ci_artifact cia on cia.ci_workflow_id = ci_workflow.id").
		Where("cia.id in (?) ", pg.In(ciArtifactIds)).
		Select()
	return ciWorkflows, err
}

func (impl *CiWorkflowRepositoryImpl) FindLastTriggeredWorkflowGitTriggersByArtifactId(ciArtifactId int) (ciWorkflow *CiWorkflow, err error) {
	workflow := &CiWorkflow{}
	err = impl.dbConnection.Model(workflow).
		Column("ci_workflow.git_triggers").
		Join("inner join ci_artifact cia on cia.ci_workflow_id = ci_workflow.id").
		Where("cia.id = ? ", ciArtifactId).
		Select()

	return workflow, err
}

func (impl *CiWorkflowRepositoryImpl) ExistsByStatus(status string) (bool, error) {
	exists, err := impl.dbConnection.Model(&CiWorkflow{}).
		Where("status =?", status).
		Exists()
	return exists, err
}

func (impl *CiWorkflowRepositoryImpl) FindBuildTypeAndStatusDataOfLast1Day() []*BuildTypeCount {
	var buildTypeCounts []*BuildTypeCount
	query := "select status,ci_build_type as type, count(*) from ci_workflow where status in ('Succeeded','Failed') and started_on > ? group by (ci_build_type, status)"
	_, err := impl.dbConnection.Query(&buildTypeCounts, query, time.Now().AddDate(0, 0, -1))
	if err != nil {
		impl.logger.Errorw("error occurred while fetching build type vs status vs count data", "err", err)
	}
	return buildTypeCounts
}

func (impl *CiWorkflowRepositoryImpl) FIndCiWorkflowStatusesByAppId(appId int) ([]*CiWorkflowStatus, error) {

	ciworkflowStatuses := make([]*CiWorkflowStatus, 0)

	query := "SELECT cw1.ci_pipeline_id,cw1.status AS ci_status,cw1.blob_storage_enabled AS storage_configured " +
		" FROM ci_workflow cw1 " +
		" INNER JOIN " +
		" (WITH cp AS (SELECT id, parent_ci_pipeline FROM ci_pipeline WHERE app_id = ? AND deleted=false ) " +
		" SELECT  cw.ci_pipeline_id, max(cw.id) " +
		" FROM ci_workflow cw WHERE cw.ci_pipeline_id IN (SELECT cp.id FROM cp) OR cw.ci_pipeline_id IN (SELECT cp.parent_ci_pipeline FROM cp) " +
		" GROUP BY ci_pipeline_id) cw2 " +
		" ON cw1.id = cw2.max;"
	_, err := impl.dbConnection.Query(&ciworkflowStatuses, query, appId) //, pg.In(ciPipelineIds))
	if err != nil {
		impl.logger.Errorw("error occurred while fetching build type vs status vs count data", "err", err)
	}
	return ciworkflowStatuses, err
}
