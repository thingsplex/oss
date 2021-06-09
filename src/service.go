package main

import (
	"flag"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/discovery"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/oss/model"
	"github.com/thingsplex/oss/router"
	"github.com/thingsplex/oss/utils"
)

func main() {
	var workDir string
	flag.StringVar(&workDir, "c", "", "Work dir")
	flag.Parse()
	if workDir == "" {
		workDir = "./"
	} else {
		fmt.Println("Work dir ", workDir)
	}
	appLifecycle := model.NewAppLifecycle()
	configs := model.NewConfigs(workDir)
	states := model.NewStates(workDir)

	err := configs.LoadFromFile()
	if err != nil {
		fmt.Print(err)
		panic("Can't load config file.")
	}
	err = states.LoadFromFile()
	if err != nil {
		fmt.Print(err)
		panic("Can't load state file")
	}

	utils.SetupLog(configs.LogFile, configs.LogLevel, configs.LogFormat)
	log.Info("--------------Starting Oss----------------")
	log.Info("Work directory : ", configs.WorkDir)
	appLifecycle.PublishEvent(model.EventConfiguring, "main", nil)

	mqtt := fimpgo.NewMqttTransport(configs.MqttServerURI, configs.MqttClientIdPrefix, configs.MqttUsername, configs.MqttPassword, true, 1, 1)
	err = mqtt.Start()
	responder := discovery.NewServiceDiscoveryResponder(mqtt)
	responder.RegisterResource(model.GetDiscoveryResource())
	responder.Start()

	fimpRouter := router.NewFromFimpRouter(mqtt, appLifecycle, configs, states)
	fimpRouter.Start()

	appLifecycle.SetConnectionState(model.ConnStateDisconnected)
	if configs.IsConfigured() && err == nil {
		appLifecycle.SetConfigState(model.ConfigStateConfigured)
		appLifecycle.SetAppState(model.AppStateRunning, nil)
		appLifecycle.SetConnectionState(model.ConnStateConnected)
	} else {
		appLifecycle.SetConfigState(model.ConfigStateNotConfigured)
		appLifecycle.SetAppState(model.AppStateNotConfigured, nil)
		appLifecycle.SetConnectionState(model.ConnStateDisconnected)
	}

	if configs.UserID != "" && configs.AccessToken != "" {
		appLifecycle.SetAuthState(model.AuthStateAuthenticated)
	} else {
		appLifecycle.SetAuthState(model.AuthStateNotAuthenticated)
	}

	mdl := model.OssHandler{}
	pollString := configs.PollTimeSec
	pollTime, err := strconv.Atoi(pollString)
	counter := 10
	var eImport float64
	var eExport float64
	// var eImportReactive float64
	// var eExportReactive float64

	log.Info("Starting signalr...")
	log.Debug("---------------------------------")
	ticker := time.NewTicker(time.Duration(pollTime) * time.Second) // Get every minute ** change to Duration(time.Minute) when ready, this is for testing.

	if !configs.IsConfigured() {
		log.Info("User needs to login and/or choose meters in playgrounds settings.")
	}

	for range ticker.C {
		if !configs.IsConfigured() {
		} else {
			counter++

			now := time.Now().UTC().Format(time.RFC3339)
			sixMinAgo := time.Now().Add(-6 * time.Minute).UTC().Format(time.RFC3339)
			fiveMinAgo := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
			oneHourAgo := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
			log.Debug("now: ", now)
			log.Debug("sixMinAgo: ", sixMinAgo)
			log.Debug("fiveMinAgo: ", fiveMinAgo)
			log.Debug("oneHourAgo: ", oneHourAgo)

			// Get hour resolution (for cumulative powers) every 5 minutes. Only updates once per hour, but not at a specific time. Get 12 times per hour to not fall too far behind.
			if counter >= 10 {
				counter = 0
				states.Telemetry, err = model.GetTelemetry(configs.AccessToken, oneHourAgo, now, configs.SelectedMeters, 2)
				if err != nil {
					log.Error("Error getting by hour resolition. Err: ", err)
				} else {
					for _, meter := range states.Telemetry {
						eImport = meter.CumulativeActivePower.Input.Max
						eExport = meter.CumulativeActivePower.Output.Max
						// eImportReactive = meter.CumulativeReactivePower.Input.Max
						// eExportReactive = meter.CumulativeReactivePower.Output.Max
					}
				}
			}

			// Get minute resolution every 30 seconds.
			states.Telemetry, err = model.GetTelemetry(configs.AccessToken, sixMinAgo, fiveMinAgo, configs.SelectedMeters, 1)
			log.Debug("meter: ", states.Telemetry)
			if err != nil {
				log.Error("Error getting by minute resolition. Err: ", err)
			} else {
				for _, meter := range states.Telemetry {
					ExtendedReportMinute := map[string]float64{
						"p_import":       math.Round(meter.ActivePower.Input.Avg*100) / 100,
						"p_export":       math.Round(meter.ActivePower.Output.Avg*100) / 100,
						"p_import_react": math.Round(meter.ReactivePower.Input.Avg*100) / 100,
						"p_export_react": math.Round(meter.ReactivePower.Output.Avg*100) / 100,
						"u1":             math.Round(meter.PhaseOne.Voltage.Avg*100) / 100,
						"u2":             math.Round(meter.PhaseTwo.Voltage.Avg*100) / 100,
						"u3":             math.Round(meter.PhaseThree.Voltage.Avg*100) / 100,
						"i1":             math.Round(meter.PhaseOne.Current.Avg*100) / 100,
						"i2":             math.Round(meter.PhaseTwo.Current.Avg*100) / 100,
						"i3":             math.Round(meter.PhaseThree.Current.Avg*100) / 100,
						"e_import":       math.Round(eImport*100) / 100,
						"e_export":       math.Round(eExport*100) / 100,
					}
					msg, adr := mdl.MakeMeterExtendedReportMsg("ossMeter", ExtendedReportMinute, nil)
					mqtt.Publish(adr, msg)
					break // Only get first meter in states.Telemetry
				}
			}
		}
		states.SaveToFile()
	}

	for {
		appLifecycle.WaitForState("main", model.AppStateRunning)
		// Configure custom resources here
		//if err := conFimpRouter.Start(); err !=nil {
		//	appLifecycle.PublishEvent(model.EventConfigError,"main",nil)
		//}else {
		//	appLifecycle.WaitForState(model.StateConfiguring,"main")
		//}
		//TODO: Add logic here
		appLifecycle.WaitForState(model.AppStateNotConfigured, "main")
	}

	mqtt.Stop()
	time.Sleep(5 * time.Second)
}
