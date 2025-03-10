package history

import (
	"encoding/json"
	repository2 "github.com/devtron-labs/devtron/internal/sql/repository"
	"github.com/devtron-labs/devtron/internal/sql/repository/chartConfig"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	chartRepoRepository "github.com/devtron-labs/devtron/pkg/chartRepo/repository"
	"github.com/devtron-labs/devtron/pkg/pipeline/history/repository"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/devtron-labs/devtron/pkg/user"
	"github.com/devtron-labs/devtron/pkg/variables"
	repository6 "github.com/devtron-labs/devtron/pkg/variables/repository"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"time"
)

type DeploymentTemplateHistoryService interface {
	CreateDeploymentTemplateHistoryFromGlobalTemplate(chart *chartRepoRepository.Chart, tx *pg.Tx, IsAppMetricsEnabled bool) error
	CreateDeploymentTemplateHistoryFromEnvOverrideTemplate(envOverride *chartConfig.EnvConfigOverride, tx *pg.Tx, IsAppMetricsEnabled bool, pipelineId int) error
	CreateDeploymentTemplateHistoryForDeploymentTrigger(pipeline *pipelineConfig.Pipeline, envOverride *chartConfig.EnvConfigOverride, renderedImageTemplate string, deployedOn time.Time, deployedBy int32) (*repository.DeploymentTemplateHistory, error)
	GetDeploymentDetailsForDeployedTemplateHistory(pipelineId, offset, limit int) ([]*DeploymentTemplateHistoryDto, error)

	GetHistoryForDeployedTemplateById(id, pipelineId int) (*HistoryDetailDto, error)
	CheckIfHistoryExistsForPipelineIdAndWfrId(pipelineId, wfrId int) (historyId int, exists bool, err error)
	GetDeployedHistoryList(pipelineId, baseConfigId int) ([]*DeployedHistoryComponentMetadataDto, error)

	// used for rollback
	GetDeployedHistoryByPipelineIdAndWfrId(pipelineId, wfrId int) (*HistoryDetailDto, error)
}

type DeploymentTemplateHistoryServiceImpl struct {
	logger                              *zap.SugaredLogger
	deploymentTemplateHistoryRepository repository.DeploymentTemplateHistoryRepository
	pipelineRepository                  pipelineConfig.PipelineRepository
	chartRepository                     chartRepoRepository.ChartRepository
	chartRefRepository                  chartRepoRepository.ChartRefRepository
	envLevelAppMetricsRepository        repository2.EnvLevelAppMetricsRepository
	appLevelMetricsRepository           repository2.AppLevelMetricsRepository
	userService                         user.UserService
	cdWorkflowRepository                pipelineConfig.CdWorkflowRepository
	variableSnapshotHistoryService      variables.VariableSnapshotHistoryService
}

func NewDeploymentTemplateHistoryServiceImpl(logger *zap.SugaredLogger, deploymentTemplateHistoryRepository repository.DeploymentTemplateHistoryRepository,
	pipelineRepository pipelineConfig.PipelineRepository,
	chartRepository chartRepoRepository.ChartRepository,
	chartRefRepository chartRepoRepository.ChartRefRepository,
	envLevelAppMetricsRepository repository2.EnvLevelAppMetricsRepository,
	appLevelMetricsRepository repository2.AppLevelMetricsRepository,
	userService user.UserService,
	cdWorkflowRepository pipelineConfig.CdWorkflowRepository,
	variableSnapshotHistoryService variables.VariableSnapshotHistoryService) *DeploymentTemplateHistoryServiceImpl {
	return &DeploymentTemplateHistoryServiceImpl{
		logger:                              logger,
		deploymentTemplateHistoryRepository: deploymentTemplateHistoryRepository,
		pipelineRepository:                  pipelineRepository,
		chartRepository:                     chartRepository,
		chartRefRepository:                  chartRefRepository,
		envLevelAppMetricsRepository:        envLevelAppMetricsRepository,
		appLevelMetricsRepository:           appLevelMetricsRepository,
		userService:                         userService,
		cdWorkflowRepository:                cdWorkflowRepository,
		variableSnapshotHistoryService:      variableSnapshotHistoryService,
	}
}

func (impl DeploymentTemplateHistoryServiceImpl) CreateDeploymentTemplateHistoryFromGlobalTemplate(chart *chartRepoRepository.Chart, tx *pg.Tx, IsAppMetricsEnabled bool) (err error) {
	//getting all pipelines without overridden charts
	pipelines, err := impl.pipelineRepository.FindAllPipelinesByChartsOverrideAndAppIdAndChartId(false, chart.AppId, chart.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("err in getting pipelines, CreateDeploymentTemplateHistoryFromGlobalTemplate", "err", err, "chart", chart)
		return err
	}
	chartRef, err := impl.chartRefRepository.FindById(chart.ChartRefId)
	if err != nil {
		impl.logger.Errorw("err in getting chartRef, CreateDeploymentTemplateHistoryFromGlobalTemplate", "err", err, "chart", chart)
		return err
	}
	if len(chartRef.Name) == 0 {
		chartRef.Name = "Rollout Deployment"
	}
	//creating history without pipeline id
	historyModel := &repository.DeploymentTemplateHistory{
		AppId:                   chart.AppId,
		ImageDescriptorTemplate: chart.ImageDescriptorTemplate,
		Template:                chart.GlobalOverride,
		Deployed:                false,
		TemplateName:            chartRef.Name,
		TemplateVersion:         chartRef.Version,
		IsAppMetricsEnabled:     IsAppMetricsEnabled,
		AuditLog: sql.AuditLog{
			CreatedOn: chart.CreatedOn,
			CreatedBy: chart.CreatedBy,
			UpdatedOn: chart.UpdatedOn,
			UpdatedBy: chart.UpdatedBy,
		},
	}
	//creating new entry
	if tx != nil {
		_, err = impl.deploymentTemplateHistoryRepository.CreateHistoryWithTxn(historyModel, tx)
	} else {
		_, err = impl.deploymentTemplateHistoryRepository.CreateHistory(historyModel)
	}
	if err != nil {
		impl.logger.Errorw("err in creating history entry for deployment template", "err", err, "history", historyModel)
		return err
	}
	for _, pipeline := range pipelines {
		historyModel := &repository.DeploymentTemplateHistory{
			AppId:                   chart.AppId,
			PipelineId:              pipeline.Id,
			ImageDescriptorTemplate: chart.ImageDescriptorTemplate,
			Template:                chart.GlobalOverride,
			Deployed:                false,
			TemplateName:            chartRef.Name,
			TemplateVersion:         chartRef.Version,
			IsAppMetricsEnabled:     IsAppMetricsEnabled,
			AuditLog: sql.AuditLog{
				CreatedOn: chart.CreatedOn,
				CreatedBy: chart.CreatedBy,
				UpdatedOn: chart.UpdatedOn,
				UpdatedBy: chart.UpdatedBy,
			},
		}
		//creating new entry
		if tx != nil {
			_, err = impl.deploymentTemplateHistoryRepository.CreateHistoryWithTxn(historyModel, tx)
		} else {
			_, err = impl.deploymentTemplateHistoryRepository.CreateHistory(historyModel)
		}
		if err != nil {
			impl.logger.Errorw("err in creating history entry for deployment template", "err", err, "history", historyModel)
			return err
		}
	}
	return err
}

func (impl DeploymentTemplateHistoryServiceImpl) CreateDeploymentTemplateHistoryFromEnvOverrideTemplate(envOverride *chartConfig.EnvConfigOverride, tx *pg.Tx, IsAppMetricsEnabled bool, pipelineId int) (err error) {
	chart, err := impl.chartRepository.FindById(envOverride.ChartId)
	if err != nil {
		impl.logger.Errorw("err in getting global deployment template", "err", err, "chart", chart)
		return err
	}
	chartRef, err := impl.chartRefRepository.FindById(chart.ChartRefId)
	if err != nil {
		impl.logger.Errorw("err in getting chartRef, CreateDeploymentTemplateHistoryFromGlobalTemplate", "err", err, "chartRef", chartRef)
		return err
	}
	if len(chartRef.Name) == 0 {
		chartRef.Name = "Rollout Deployment"
	}
	if pipelineId == 0 {
		pipeline, err := impl.pipelineRepository.GetByEnvOverrideIdAndEnvId(envOverride.Id, envOverride.TargetEnvironment)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("err in getting pipelines, CreateDeploymentTemplateHistoryFromEnvOverrideTemplate", "err", err, "envOverrideId", envOverride.Id)
			return err
		}
		pipelineId = pipeline.Id
	}
	historyModel := &repository.DeploymentTemplateHistory{
		AppId:                   chart.AppId,
		PipelineId:              pipelineId,
		ImageDescriptorTemplate: chart.ImageDescriptorTemplate,
		TargetEnvironment:       envOverride.TargetEnvironment,
		Deployed:                false,
		TemplateName:            chartRef.Name,
		TemplateVersion:         chartRef.Version,
		IsAppMetricsEnabled:     IsAppMetricsEnabled,
		AuditLog: sql.AuditLog{
			CreatedOn: envOverride.CreatedOn,
			CreatedBy: envOverride.CreatedBy,
			UpdatedOn: envOverride.UpdatedOn,
			UpdatedBy: envOverride.UpdatedBy,
		},
	}
	if envOverride.IsOverride {
		historyModel.Template = envOverride.EnvOverrideValues
	} else {
		//this is for the case when env override is created for new cd pipelines with template = "{}"
		historyModel.Template = chart.GlobalOverride
	}
	//creating new entry
	if tx != nil {
		_, err = impl.deploymentTemplateHistoryRepository.CreateHistoryWithTxn(historyModel, tx)
	} else {
		_, err = impl.deploymentTemplateHistoryRepository.CreateHistory(historyModel)
	}
	if err != nil {
		impl.logger.Errorw("err in creating history entry for deployment template", "err", err, "history", historyModel)
		return err
	}
	return nil
}

func (impl DeploymentTemplateHistoryServiceImpl) CreateDeploymentTemplateHistoryForDeploymentTrigger(pipeline *pipelineConfig.Pipeline, envOverride *chartConfig.EnvConfigOverride, renderedImageTemplate string, deployedOn time.Time, deployedBy int32) (*repository.DeploymentTemplateHistory, error) {
	chartRef, err := impl.chartRefRepository.FindById(envOverride.Chart.ChartRefId)
	if err != nil {
		impl.logger.Errorw("err in getting chartRef, CreateDeploymentTemplateHistoryFromGlobalTemplate", "err", err, "chartRef", chartRef)
		return nil, err
	}
	if len(chartRef.Name) == 0 {
		chartRef.Name = "Rollout Deployment"
	}
	isAppMetricsEnabled := false
	envLevelAppMetrics, err := impl.envLevelAppMetricsRepository.FindByAppIdAndEnvId(pipeline.AppId, pipeline.EnvironmentId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting env level app metrics", "err", err, "appId", pipeline.AppId, "envId", pipeline.EnvironmentId)
		return nil, err
	} else if err == pg.ErrNoRows {
		appLevelAppMetrics, err := impl.appLevelMetricsRepository.FindByAppId(pipeline.AppId)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in getting app level app metrics", "err", err, "appId", pipeline.AppId)
			return nil, err
		} else if err == nil {
			isAppMetricsEnabled = appLevelAppMetrics.AppMetrics
		}
	} else {
		isAppMetricsEnabled = *envLevelAppMetrics.AppMetrics
	}
	historyModel := &repository.DeploymentTemplateHistory{
		AppId:                   pipeline.AppId,
		PipelineId:              pipeline.Id,
		TargetEnvironment:       pipeline.EnvironmentId,
		ImageDescriptorTemplate: renderedImageTemplate,
		Deployed:                true,
		DeployedBy:              deployedBy,
		DeployedOn:              deployedOn,
		TemplateName:            chartRef.Name,
		TemplateVersion:         chartRef.Version,
		IsAppMetricsEnabled:     isAppMetricsEnabled,
		AuditLog: sql.AuditLog{
			CreatedOn: deployedOn,
			CreatedBy: deployedBy,
			UpdatedOn: deployedOn,
			UpdatedBy: deployedBy,
		},
	}
	if envOverride.IsOverride {
		historyModel.Template = envOverride.EnvOverrideValues
	} else {
		historyModel.Template = envOverride.Chart.GlobalOverride
	}
	//creating new entry
	history, err := impl.deploymentTemplateHistoryRepository.CreateHistory(historyModel)
	if err != nil {
		impl.logger.Errorw("err in creating history entry for deployment template", "err", err, "history", historyModel)
		return nil, err
	}
	return history, nil
}

func (impl DeploymentTemplateHistoryServiceImpl) GetDeploymentDetailsForDeployedTemplateHistory(pipelineId, offset, limit int) ([]*DeploymentTemplateHistoryDto, error) {
	histories, err := impl.deploymentTemplateHistoryRepository.GetDeploymentDetailsForDeployedTemplateHistory(pipelineId, offset, limit)
	if err != nil {
		impl.logger.Errorw("error in getting deployment template history", "err", err, "pipelineId", pipelineId)
		return nil, err
	}
	//getting wfrList for status of history
	wfrList, err := impl.cdWorkflowRepository.FindCdWorkflowMetaByPipelineId(pipelineId, offset, limit)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting ")
		return nil, err
	}
	deploymentTimeStatusMap := make(map[time.Time]int)
	for index, wfr := range wfrList {
		deploymentTimeStatusMap[wfr.StartedOn] = index
	}
	var historiesDto []*DeploymentTemplateHistoryDto
	for _, history := range histories {
		if wfrIndex, ok := deploymentTimeStatusMap[history.DeployedOn]; ok {
			user, err := impl.userService.GetById(history.DeployedBy)
			if err != nil {
				impl.logger.Errorw("unable to find user by id", "err", err, "id", history.Id)
				return nil, err
			}
			historyDto := &DeploymentTemplateHistoryDto{
				Id:               history.Id,
				AppId:            history.AppId,
				PipelineId:       history.PipelineId,
				Deployed:         history.Deployed,
				DeployedOn:       history.DeployedOn,
				DeployedBy:       history.DeployedBy,
				EmailId:          user.EmailId,
				DeploymentStatus: wfrList[wfrIndex].Status,
				WfrId:            wfrList[wfrIndex].Id,
				WorkflowType:     string(wfrList[wfrIndex].WorkflowType),
			}
			historiesDto = append(historiesDto, historyDto)
		}
	}
	return historiesDto, nil
}

func (impl DeploymentTemplateHistoryServiceImpl) CheckIfHistoryExistsForPipelineIdAndWfrId(pipelineId, wfrId int) (historyId int, exists bool, err error) {
	impl.logger.Debugw("received request, CheckIfHistoryExistsForPipelineIdAndWfrId", "pipelineId", pipelineId, "wfrId", wfrId)

	//checking if history exists for pipelineId and wfrId
	history, err := impl.deploymentTemplateHistoryRepository.GetHistoryByPipelineIdAndWfrId(pipelineId, wfrId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in checking if history exists for pipelineId and wfrId", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return 0, false, err
	} else if err == pg.ErrNoRows {
		return 0, false, nil
	}
	return history.Id, true, nil
}

func (impl DeploymentTemplateHistoryServiceImpl) GetDeployedHistoryByPipelineIdAndWfrId(pipelineId, wfrId int) (*HistoryDetailDto, error) {
	impl.logger.Debugw("received request, GetDeployedHistoryByPipelineIdAndWfrId", "pipelineId", pipelineId, "wfrId", wfrId)

	//checking if history exists for pipelineId and wfrId
	history, err := impl.deploymentTemplateHistoryRepository.GetHistoryByPipelineIdAndWfrId(pipelineId, wfrId)
	if err != nil {
		impl.logger.Errorw("error in checking if history exists for pipelineId and wfrId", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return nil, err
	}

	variableSnapshotMap, err := impl.getVariableSnapshot(history.Id)
	if err != nil {
		return nil, err
	}

	historyDto := &HistoryDetailDto{
		TemplateName:        history.TemplateName,
		TemplateVersion:     history.TemplateVersion,
		IsAppMetricsEnabled: &history.IsAppMetricsEnabled,
		CodeEditorValue: &HistoryDetailConfig{
			DisplayName: "values.yaml",
			Value:       history.Template,
		},
		VariableSnapshot: variableSnapshotMap,
	}
	return historyDto, nil
}

func (impl DeploymentTemplateHistoryServiceImpl) getVariableSnapshot(historyId int) (map[string]string, error) {
	reference := repository6.HistoryReference{
		HistoryReferenceId:   historyId,
		HistoryReferenceType: repository6.HistoryReferenceTypeDeploymentTemplate,
	}
	references, err := impl.variableSnapshotHistoryService.GetVariableHistoryForReferences([]repository6.HistoryReference{reference})
	if err != nil {
		return nil, err
	}
	variableSnapshotMap := make(map[string]string)
	if _, ok := references[reference]; ok {
		err = json.Unmarshal(references[reference].VariableSnapshot, &variableSnapshotMap)
		if err != nil {
			return nil, err
		}
	}
	return variableSnapshotMap, nil
}

func (impl DeploymentTemplateHistoryServiceImpl) GetDeployedHistoryList(pipelineId, baseConfigId int) ([]*DeployedHistoryComponentMetadataDto, error) {
	impl.logger.Debugw("received request, GetDeployedHistoryList", "pipelineId", pipelineId, "baseConfigId", baseConfigId)

	//checking if history exists for pipelineId and wfrId
	histories, err := impl.deploymentTemplateHistoryRepository.GetDeployedHistoryList(pipelineId, baseConfigId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting history list for pipelineId and baseConfigId", "err", err, "pipelineId", pipelineId)
		return nil, err
	}
	var historyList []*DeployedHistoryComponentMetadataDto
	for _, history := range histories {
		historyList = append(historyList, &DeployedHistoryComponentMetadataDto{
			Id:               history.Id,
			DeployedOn:       history.DeployedOn,
			DeployedBy:       history.DeployedByEmailId,
			DeploymentStatus: history.DeploymentStatus,
		})
	}
	return historyList, nil
}

func (impl DeploymentTemplateHistoryServiceImpl) GetHistoryForDeployedTemplateById(id, pipelineId int) (*HistoryDetailDto, error) {
	history, err := impl.deploymentTemplateHistoryRepository.GetHistoryForDeployedTemplateById(id, pipelineId)
	if err != nil {
		impl.logger.Errorw("error in getting deployment template history", "err", err, "id", id, "pipelineId", pipelineId)
		return nil, err
	}

	variableSnapshotMap, err := impl.getVariableSnapshot(history.Id)
	if err != nil {
		return nil, err
	}
	historyDto := &HistoryDetailDto{
		TemplateName:        history.TemplateName,
		TemplateVersion:     history.TemplateVersion,
		IsAppMetricsEnabled: &history.IsAppMetricsEnabled,
		CodeEditorValue: &HistoryDetailConfig{
			DisplayName: "values.yaml",
			Value:       history.Template,
		},
		VariableSnapshot: variableSnapshotMap,
	}
	return historyDto, nil
}
