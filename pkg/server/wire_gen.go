// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire gen -tags "enterprise"
//+build !wireinject

package server

import (
	"github.com/google/wire"
	httpclient2 "github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana/pkg/api"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/extensions/accesscontrol/database"
	manager3 "github.com/grafana/grafana/pkg/extensions/accesscontrol/manager"
	"github.com/grafana/grafana/pkg/extensions/accesscontrol/provisioner"
	"github.com/grafana/grafana/pkg/extensions/licensing"
	"github.com/grafana/grafana/pkg/extensions/provisioning/service"
	"github.com/grafana/grafana/pkg/extensions/requestinterceptor"
	"github.com/grafana/grafana/pkg/extensions/settings/settingsprovider"
	"github.com/grafana/grafana/pkg/infra/backgroundsvcs"
	"github.com/grafana/grafana/pkg/infra/httpclient"
	"github.com/grafana/grafana/pkg/infra/httpclient/httpclientprovider"
	"github.com/grafana/grafana/pkg/infra/localcache"
	metrics2 "github.com/grafana/grafana/pkg/infra/metrics"
	"github.com/grafana/grafana/pkg/infra/remotecache"
	"github.com/grafana/grafana/pkg/infra/serverlock"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/infra/usagestats"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/plugins/backendplugin"
	"github.com/grafana/grafana/pkg/plugins/backendplugin/manager"
	manager2 "github.com/grafana/grafana/pkg/plugins/manager"
	"github.com/grafana/grafana/pkg/plugins/plugincontext"
	"github.com/grafana/grafana/pkg/plugins/plugindashboards"
	"github.com/grafana/grafana/pkg/services/alerting"
	"github.com/grafana/grafana/pkg/services/auth"
	"github.com/grafana/grafana/pkg/services/auth/jwt"
	"github.com/grafana/grafana/pkg/services/cleanup"
	"github.com/grafana/grafana/pkg/services/contexthandler"
	"github.com/grafana/grafana/pkg/services/datasourceproxy"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/services/hooks"
	"github.com/grafana/grafana/pkg/services/libraryelements"
	"github.com/grafana/grafana/pkg/services/librarypanels"
	"github.com/grafana/grafana/pkg/services/live"
	"github.com/grafana/grafana/pkg/services/live/pushhttp"
	"github.com/grafana/grafana/pkg/services/login"
	"github.com/grafana/grafana/pkg/services/login/loginservice"
	"github.com/grafana/grafana/pkg/services/ngalert"
	"github.com/grafana/grafana/pkg/services/ngalert/metrics"
	"github.com/grafana/grafana/pkg/services/notifications"
	"github.com/grafana/grafana/pkg/services/quota"
	"github.com/grafana/grafana/pkg/services/rendering"
	"github.com/grafana/grafana/pkg/services/schemaloader"
	"github.com/grafana/grafana/pkg/services/search"
	"github.com/grafana/grafana/pkg/services/shorturls"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tsdb"
	"github.com/grafana/grafana/pkg/tsdb/azuremonitor"
	"github.com/grafana/grafana/pkg/tsdb/cloudmonitoring"
	"github.com/grafana/grafana/pkg/tsdb/cloudwatch"
	"github.com/grafana/grafana/pkg/tsdb/postgres"
	"github.com/grafana/grafana/pkg/tsdb/testdatasource"
)

import (
	_ "github.com/grafana/grafana/pkg/extensions"
)

// Injectors from wire.go:

func Initialize(cla setting.CommandLineArgs, opts Options, apiOpts api.ServerOptions) (*Server, error) {
	cfg, err := setting.NewCfgFromArgs(cla)
	if err != nil {
		return nil, err
	}
	cacheService := localcache.ProvideService()
	inProcBus := bus.ProvideBus()
	sqlStore, err := sqlstore.ProvideService(cfg, cacheService, inProcBus)
	if err != nil {
		return nil, err
	}
	routeRegisterImpl := routing.ProvideRegister(cfg)
	container := backgroundsvcs.ProvideService()
	remoteCache, err := remotecache.ProvideService(cfg, sqlStore, container)
	if err != nil {
		return nil, err
	}
	hooksService := hooks.ProvideService()
	licensingLicensing := licensing.ProvideLicensing(hooksService, cfg, inProcBus, sqlStore, routeRegisterImpl)
	licenseTokenService := licensing.ProvideLicenseTokenService(licensingLicensing, routeRegisterImpl, cfg, sqlStore)
	requestInterceptor := requestinterceptor.ProvideService(cfg)
	managerManager := manager.ProvideService(cfg, licenseTokenService, requestInterceptor, container)
	pluginManager, err := manager2.ProvideService(cfg, sqlStore, managerManager, container)
	if err != nil {
		return nil, err
	}
	renderingService, err := rendering.ProvideService(cfg, remoteCache, pluginManager, container)
	if err != nil {
		return nil, err
	}
	logsService := cloudwatch.ProvideLogsService()
	cloudWatchService := cloudwatch.ProvideService(cfg, logsService, managerManager)
	cloudmonitoringService := cloudmonitoring.ProvideService(pluginManager)
	azuremonitorService := azuremonitor.ProvideService(pluginManager)
	postgresService := postgres.ProvideService(cfg)
	provider := httpclientprovider.New(cfg)
	testDataPlugin, err := testdatasource.ProvideService(cfg, managerManager)
	if err != nil {
		return nil, err
	}
	tsdbService := tsdb.NewService(cfg, cloudWatchService, cloudmonitoringService, azuremonitorService, pluginManager, postgresService, provider, testDataPlugin, managerManager)
	alertEngine := alerting.ProvideAlertEngine(renderingService, inProcBus, requestInterceptor, tsdbService, container, cfg)
	usageStatsService := usagestats.ProvideService(cfg, inProcBus, sqlStore, alertEngine, licenseTokenService, pluginManager, container)
	implementation, err := settingsprovider.ProvideService(cfg, sqlStore, routeRegisterImpl)
	if err != nil {
		return nil, err
	}
	cacheServiceImpl := datasources.ProvideCacheService(cacheService, sqlStore)
	serverLockService := serverlock.ProvideService(sqlStore)
	userAuthTokenService := auth.ProvideUserAuthTokenService(sqlStore, serverLockService, cfg, container)
	shortURLService := shorturls.ProvideService(sqlStore)
	cleanUpService := cleanup.ProvideService(cfg, serverLockService, shortURLService, container)
	accessControlStore := database.ProvideService(sqlStore)
	cacheWrapper, err := manager3.ProvideService(cfg, routeRegisterImpl, accessControlStore, licenseTokenService, usageStatsService)
	if err != nil {
		return nil, err
	}
	accessControlProvisionerImpl := provisioner.ProvideService(cfg, accessControlStore)
	enterpriseProvisioningServiceImpl, err := service.ProvideService(cfg, cacheWrapper, licenseTokenService, sqlStore, pluginManager, routeRegisterImpl, accessControlProvisionerImpl, container)
	if err != nil {
		return nil, err
	}
	quotaService := quota.ProvideService(cfg, userAuthTokenService)
	loginserviceImplementation := loginservice.ProvideService(sqlStore, inProcBus, quotaService)
	dataSourceProxyService := datasourceproxy.ProvideService(cacheServiceImpl, requestInterceptor, pluginManager, cfg, provider)
	searchService := search.ProvideService(cfg, inProcBus)
	plugincontextProvider := plugincontext.ProvideService(inProcBus, cacheService, pluginManager, cacheServiceImpl)
	grafanaLive, err := live.ProvideService(plugincontextProvider, cfg, routeRegisterImpl, logsService, pluginManager, cacheService, cacheServiceImpl, sqlStore, container)
	if err != nil {
		return nil, err
	}
	gateway := pushhttp.ProvideService(cfg, grafanaLive, container)
	authService, err := jwt.ProvideService(cfg, remoteCache)
	if err != nil {
		return nil, err
	}
	contextHandler := contexthandler.ProvideService(cfg, userAuthTokenService, authService, remoteCache, renderingService, sqlStore)
	plugindashboardsService := plugindashboards.ProvideService(tsdbService, pluginManager, sqlStore, container)
	schemaLoaderService, err := schemaloader.ProvideService(cfg)
	if err != nil {
		return nil, err
	}
	metricsMetrics := metrics.ProvideService()
	alertNG, err := ngalert.ProvideService(cfg, cacheServiceImpl, routeRegisterImpl, sqlStore, tsdbService, dataSourceProxyService, quotaService, container, metricsMetrics)
	if err != nil {
		return nil, err
	}
	libraryElementService := libraryelements.ProvideService(cfg, sqlStore, routeRegisterImpl)
	libraryPanelService := librarypanels.ProvideService(cfg, sqlStore, routeRegisterImpl, libraryElementService)
	notificationService, err := notifications.ProvideService(inProcBus, cfg, container)
	if err != nil {
		return nil, err
	}
	tracingService, err := tracing.ProvideService(cfg, container)
	if err != nil {
		return nil, err
	}
	internalMetricsService, err := metrics2.ProvideService(cfg, container)
	if err != nil {
		return nil, err
	}
	httpServer := api.ProvideHTTPServer(apiOpts, cfg, routeRegisterImpl, inProcBus, renderingService, licenseTokenService, hooksService, cacheService, sqlStore, tsdbService, alertEngine, usageStatsService, requestInterceptor, pluginManager, managerManager, implementation, cacheServiceImpl, userAuthTokenService, cleanUpService, shortURLService, remoteCache, enterpriseProvisioningServiceImpl, loginserviceImplementation, cacheWrapper, dataSourceProxyService, searchService, grafanaLive, gateway, plugincontextProvider, contextHandler, plugindashboardsService, schemaLoaderService, alertNG, libraryPanelService, libraryElementService, notificationService, tracingService, internalMetricsService, quotaService, container)
	server, err := New(opts, cfg, sqlStore, httpServer, enterpriseProvisioningServiceImpl, container)
	if err != nil {
		return nil, err
	}
	return server, nil
}

func InitializeForTest(cla setting.CommandLineArgs, opts Options, apiOpts api.ServerOptions, sqlStore *sqlstore.SQLStore) (*Server, error) {
	cfg, err := setting.NewCfgFromArgs(cla)
	if err != nil {
		return nil, err
	}
	routeRegisterImpl := routing.ProvideRegister(cfg)
	inProcBus := bus.ProvideBus()
	container := backgroundsvcs.ProvideService()
	remoteCache, err := remotecache.ProvideService(cfg, sqlStore, container)
	if err != nil {
		return nil, err
	}
	hooksService := hooks.ProvideService()
	licensingLicensing := licensing.ProvideLicensing(hooksService, cfg, inProcBus, sqlStore, routeRegisterImpl)
	licenseTokenService := licensing.ProvideLicenseTokenService(licensingLicensing, routeRegisterImpl, cfg, sqlStore)
	requestInterceptor := requestinterceptor.ProvideService(cfg)
	managerManager := manager.ProvideService(cfg, licenseTokenService, requestInterceptor, container)
	pluginManager, err := manager2.ProvideService(cfg, sqlStore, managerManager, container)
	if err != nil {
		return nil, err
	}
	renderingService, err := rendering.ProvideService(cfg, remoteCache, pluginManager, container)
	if err != nil {
		return nil, err
	}
	cacheService := localcache.ProvideService()
	logsService := cloudwatch.ProvideLogsService()
	cloudWatchService := cloudwatch.ProvideService(cfg, logsService, managerManager)
	cloudmonitoringService := cloudmonitoring.ProvideService(pluginManager)
	azuremonitorService := azuremonitor.ProvideService(pluginManager)
	postgresService := postgres.ProvideService(cfg)
	provider := httpclientprovider.New(cfg)
	testDataPlugin, err := testdatasource.ProvideService(cfg, managerManager)
	if err != nil {
		return nil, err
	}
	tsdbService := tsdb.NewService(cfg, cloudWatchService, cloudmonitoringService, azuremonitorService, pluginManager, postgresService, provider, testDataPlugin, managerManager)
	alertEngine := alerting.ProvideAlertEngine(renderingService, inProcBus, requestInterceptor, tsdbService, container, cfg)
	usageStatsService := usagestats.ProvideService(cfg, inProcBus, sqlStore, alertEngine, licenseTokenService, pluginManager, container)
	implementation, err := settingsprovider.ProvideService(cfg, sqlStore, routeRegisterImpl)
	if err != nil {
		return nil, err
	}
	cacheServiceImpl := datasources.ProvideCacheService(cacheService, sqlStore)
	serverLockService := serverlock.ProvideService(sqlStore)
	userAuthTokenService := auth.ProvideUserAuthTokenService(sqlStore, serverLockService, cfg, container)
	shortURLService := shorturls.ProvideService(sqlStore)
	cleanUpService := cleanup.ProvideService(cfg, serverLockService, shortURLService, container)
	accessControlStore := database.ProvideService(sqlStore)
	cacheWrapper, err := manager3.ProvideService(cfg, routeRegisterImpl, accessControlStore, licenseTokenService, usageStatsService)
	if err != nil {
		return nil, err
	}
	accessControlProvisionerImpl := provisioner.ProvideService(cfg, accessControlStore)
	enterpriseProvisioningServiceImpl, err := service.ProvideService(cfg, cacheWrapper, licenseTokenService, sqlStore, pluginManager, routeRegisterImpl, accessControlProvisionerImpl, container)
	if err != nil {
		return nil, err
	}
	quotaService := quota.ProvideService(cfg, userAuthTokenService)
	loginserviceImplementation := loginservice.ProvideService(sqlStore, inProcBus, quotaService)
	dataSourceProxyService := datasourceproxy.ProvideService(cacheServiceImpl, requestInterceptor, pluginManager, cfg, provider)
	searchService := search.ProvideService(cfg, inProcBus)
	plugincontextProvider := plugincontext.ProvideService(inProcBus, cacheService, pluginManager, cacheServiceImpl)
	grafanaLive, err := live.ProvideService(plugincontextProvider, cfg, routeRegisterImpl, logsService, pluginManager, cacheService, cacheServiceImpl, sqlStore, container)
	if err != nil {
		return nil, err
	}
	gateway := pushhttp.ProvideService(cfg, grafanaLive, container)
	authService, err := jwt.ProvideService(cfg, remoteCache)
	if err != nil {
		return nil, err
	}
	contextHandler := contexthandler.ProvideService(cfg, userAuthTokenService, authService, remoteCache, renderingService, sqlStore)
	plugindashboardsService := plugindashboards.ProvideService(tsdbService, pluginManager, sqlStore, container)
	schemaLoaderService, err := schemaloader.ProvideService(cfg)
	if err != nil {
		return nil, err
	}
	metricsMetrics := metrics.ProvideServiceForTest()
	alertNG, err := ngalert.ProvideService(cfg, cacheServiceImpl, routeRegisterImpl, sqlStore, tsdbService, dataSourceProxyService, quotaService, container, metricsMetrics)
	if err != nil {
		return nil, err
	}
	libraryElementService := libraryelements.ProvideService(cfg, sqlStore, routeRegisterImpl)
	libraryPanelService := librarypanels.ProvideService(cfg, sqlStore, routeRegisterImpl, libraryElementService)
	notificationService, err := notifications.ProvideService(inProcBus, cfg, container)
	if err != nil {
		return nil, err
	}
	tracingService, err := tracing.ProvideService(cfg, container)
	if err != nil {
		return nil, err
	}
	internalMetricsService, err := metrics2.ProvideService(cfg, container)
	if err != nil {
		return nil, err
	}
	httpServer := api.ProvideHTTPServer(apiOpts, cfg, routeRegisterImpl, inProcBus, renderingService, licenseTokenService, hooksService, cacheService, sqlStore, tsdbService, alertEngine, usageStatsService, requestInterceptor, pluginManager, managerManager, implementation, cacheServiceImpl, userAuthTokenService, cleanUpService, shortURLService, remoteCache, enterpriseProvisioningServiceImpl, loginserviceImplementation, cacheWrapper, dataSourceProxyService, searchService, grafanaLive, gateway, plugincontextProvider, contextHandler, plugindashboardsService, schemaLoaderService, alertNG, libraryPanelService, libraryElementService, notificationService, tracingService, internalMetricsService, quotaService, container)
	server, err := New(opts, cfg, sqlStore, httpServer, enterpriseProvisioningServiceImpl, container)
	if err != nil {
		return nil, err
	}
	return server, nil
}

// wire.go:

var wireBasicSet = wire.NewSet(tsdb.NewService, wire.Bind(new(plugins.DataRequestHandler), new(*tsdb.Service)), alerting.ProvideAlertEngine, wire.Bind(new(alerting.UsageStatsQuerier), new(*alerting.AlertEngine)), setting.NewCfgFromArgs, New, api.ProvideHTTPServer, bus.ProvideBus, wire.Bind(new(bus.Bus), new(*bus.InProcBus)), rendering.ProvideService, wire.Bind(new(rendering.Service), new(*rendering.RenderingService)), routing.ProvideRegister, wire.Bind(new(routing.RouteRegister), new(*routing.RouteRegisterImpl)), hooks.ProvideService, localcache.ProvideService, usagestats.ProvideService, wire.Bind(new(usagestats.UsageStats), new(*usagestats.UsageStatsService)), manager2.ProvideService, wire.Bind(new(plugins.Manager), new(*manager2.PluginManager)), manager.ProvideService, wire.Bind(new(backendplugin.Manager), new(*manager.Manager)), cloudwatch.ProvideService, cloudwatch.ProvideLogsService, cloudmonitoring.ProvideService, azuremonitor.ProvideService, postgres.ProvideService, httpclientprovider.New, wire.Bind(new(httpclient.Provider), new(*httpclient2.Provider)), datasources.ProvideCacheService, wire.Bind(new(datasources.CacheService), new(*datasources.CacheServiceImpl)), auth.ProvideUserAuthTokenService, wire.Bind(new(models.UserTokenService), new(*auth.UserAuthTokenService)), serverlock.ProvideService, cleanup.ProvideService, shorturls.ProvideService, wire.Bind(new(shorturls.Service), new(*shorturls.ShortURLService)), quota.ProvideService, remotecache.ProvideService, loginservice.ProvideService, wire.Bind(new(login.Service), new(*loginservice.Implementation)), datasourceproxy.ProvideService, search.ProvideService, live.ProvideService, pushhttp.ProvideService, plugincontext.ProvideService, contexthandler.ProvideService, jwt.ProvideService, wire.Bind(new(models.JWTService), new(*jwt.AuthService)), plugindashboards.ProvideService, schemaloader.ProvideService, ngalert.ProvideService, librarypanels.ProvideService, wire.Bind(new(librarypanels.Service), new(*librarypanels.LibraryPanelService)), libraryelements.ProvideService, wire.Bind(new(libraryelements.Service), new(*libraryelements.LibraryElementService)), notifications.ProvideService, tracing.ProvideService, metrics2.ProvideService, backgroundsvcs.ProvideService, testdatasource.ProvideService)

var wireSet = wire.NewSet(
	wireBasicSet, sqlstore.ProvideService, metrics.ProvideService,
)

var wireTestSet = wire.NewSet(
	wireBasicSet, metrics.ProvideServiceForTest,
)
