package router

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/oss/model"
)

type FromFimpRouter struct {
	inboundMsgCh fimpgo.MessageCh
	mqt          *fimpgo.MqttTransport
	instanceId   string
	appLifecycle *model.Lifecycle
	configs      *model.Configs
	states       *model.States
	resetToken   *model.ResetToken
	auth         *model.Auth
}

func NewFromFimpRouter(mqt *fimpgo.MqttTransport, appLifecycle *model.Lifecycle, configs *model.Configs, states *model.States) *FromFimpRouter {
	fc := FromFimpRouter{inboundMsgCh: make(fimpgo.MessageCh, 5), mqt: mqt, appLifecycle: appLifecycle, configs: configs, states: states}
	fc.mqt.RegisterChannel("ch1", fc.inboundMsgCh)
	return &fc
}

func (fc *FromFimpRouter) Start() {

	// TODO: Choose either adapter or app topic

	// ------ Adapter topics ---------------------------------------------
	fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:dev/rn:%s/ad:1/#", model.ServiceName))
	fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:ad/rn:%s/ad:1", model.ServiceName))

	// ------ Application topic -------------------------------------------
	//fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:app/rn:%s/ad:1",model.ServiceName))

	go func(msgChan fimpgo.MessageCh) {
		for {
			select {
			case newMsg := <-msgChan:
				fc.routeFimpMessage(newMsg)
			}
		}
	}(fc.inboundMsgCh)
}

func (fc *FromFimpRouter) routeFimpMessage(newMsg *fimpgo.Message) {
	ns := model.NetworkService{}
	log.Debug("New fimp msg with service '", newMsg.Payload.Service, "' and type '", newMsg.Payload.Type, "'.")
	addr := strings.Replace(newMsg.Addr.ServiceAddress, "_0", "", 1)
	switch newMsg.Payload.Service {
	case "chargepoint":
		addr = strings.Replace(addr, "l", "", 1)
		switch newMsg.Payload.Type {
		case "cmd.charge.start":
			// get address
		}

	case model.ServiceName:
		adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
		switch newMsg.Payload.Type {

		case "cmd.app.get_manifest":
			mode, err := newMsg.Payload.GetStringValue()
			if err != nil {
				log.Error("Incorrect request format ")
				return
			}
			manifest := model.NewManifest()
			err = manifest.LoadFromFile(filepath.Join(fc.configs.GetDefaultDir(), "app-manifest.json"))
			if err != nil {
				log.Error("Failed to load manifest file .Error :", err.Error())
				return
			}
			if mode == "manifest_state" {
				manifest.AppState = *fc.appLifecycle.GetAllStates()
				manifest.ConfigState = fc.configs
			}
			if fc.configs.AccessToken != "" {
				if fc.states.IsConfigured() {
					var meterSelect []interface{}
					manifest.UIBlocks[0].Hidden = false
					manifest.Configs[0].ValT = "str_map"
					manifest.Configs[0].UI.Type = "list_radio"
					for _, meters := range fc.states.Meters.Meters {
						MeterName := fmt.Sprintf("%v", meters.MeterNumber)
						MeterNameUser := fmt.Sprintf("%v, %s, %s %s", meters.MeterNumber[len(meters.MeterNumber)-5:], meters.MeterAddress.StreetAddress1, meters.MeterAddress.PostCode, meters.MeterAddress.PostalArea)
						meterSelect = append(meterSelect, map[string]interface{}{"val": MeterName, "label": map[string]interface{}{"en": MeterNameUser}})
					}

					manifest.Configs[0].UI.Select = meterSelect
				} else {
					manifest.UIBlocks[0].Hidden = true
					manifest.Configs[0].ValT = "string"
					manifest.Configs[0].UI.Type = "input_readonly"
					manifest.Configs[0].UI.Select = nil
					var val model.Value
					val.Default = "Please refresh this page..."
					manifest.Configs[0].Val = val
				}
			} else {
				manifest.UIBlocks[0].Hidden = true
				manifest.Configs[0].ValT = "string"
				manifest.Configs[0].UI.Type = "input_readonly"
				manifest.Configs[0].UI.Select = nil
				var val model.Value
				val.Default = "You need to login first"
				manifest.Configs[0].Val = val
			}
			if fc.configs.Email != "" || fc.configs.EmailCode != "" {
				manifest.UIBlocks[3].Hidden = false
			} else {
				manifest.UIBlocks[3].Hidden = true
			}
			if fc.configs.Email == "" {
				// user needs to input email
				manifest.UIBlocks[1].Hidden = false // email block
				manifest.UIBlocks[2].Hidden = true  // email code block
			}
			if fc.configs.Email != "" && fc.configs.EmailCode == "" {
				// user needs to input code
				manifest.UIBlocks[1].Hidden = false
				manifest.UIBlocks[2].Hidden = false
			}
			msg := fimpgo.NewMessage("evt.app.manifest_report", model.ServiceName, fimpgo.VTypeObject, manifest, nil, nil, newMsg.Payload)
			msg.Source = "oss"
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.app.get_state":
			msg := fimpgo.NewMessage("evt.app.manifest_report", model.ServiceName, fimpgo.VTypeObject, fc.appLifecycle.GetAllStates(), nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.config.get_extended_report":

			msg := fimpgo.NewMessage("evt.config.extended_report", model.ServiceName, fimpgo.VTypeObject, fc.configs, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.config.extended_set":
			conf := model.Configs{}
			err := newMsg.Payload.GetObjectValue(&conf)
			opStatus := "ok"
			errorText := ""
			if err != nil {
				log.Error("Can't parse configuration object")
				return
			}

			if fc.configs.Email != conf.Email {
				log.Debug("Old email: ", fc.configs.Email)
				log.Info("New email: ", conf.Email)
				fc.configs.Email = strings.TrimSpace(conf.Email)
				fc.configs.UserID, err = fc.auth.GetAuthCode(fc.configs.Email)
				if err != nil {
					opStatus = "error"
					errorText = "Wrong or invalid email address."
					log.Error(err)
				} else {
					opStatus = "ok"
					errorText = ""
				}
			}

			if fc.configs.EmailCode != conf.EmailCode {
				log.Debug("Old emailCode: ", fc.configs.EmailCode)
				log.Info("New emailCode: ", conf.EmailCode)
				fc.configs.EmailCode = conf.EmailCode
				fc.configs.AccessToken, err = fc.auth.GetAuthToken(fc.configs.UserID, fc.configs.EmailCode)
				if err != nil {
					opStatus = "error"
					errorText = "Wrong or invalid code."
					log.Error("Error: ", err)
				} else {
					opStatus = "ok"
					errorText = ""
				}

				log.Debug("AccessToken: ", fc.configs.AccessToken)

				log.Debug("Getting meters")
				fc.states.Meters, err = model.GetMeters(fc.configs.AccessToken)
				log.Info("Meters: ", fc.states.Meters)
			}

			fc.configs.SelectedMeters = conf.SelectedMeters
			if len(fc.configs.SelectedMeters) != 0 {
				fc.appLifecycle.SetConfigState(model.ConfigStateConfigured)
				fc.appLifecycle.SetConnectionState(model.ConnStateConnected)
				fc.appLifecycle.SetAppState(model.AppStateRunning, nil)
				fc.appLifecycle.SetAuthState(model.AuthStateAuthenticated)
				log.Debug("Selected meter: ", fc.configs.SelectedMeters)
				inclReport := ns.MakeInclusionReport("ossMeter")
				msg := fimpgo.NewMessage("evt.thing.inclusion_report", "oss", "object", inclReport, nil, nil, newMsg.Payload)
				adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "oss", ResourceAddress: "1"}
				fc.mqt.Publish(&adr, msg)
				log.Info("Inclusion report sent")
			} else {
				fc.appLifecycle.SetConfigState(model.ConfigStateNotConfigured)
				fc.appLifecycle.SetConnectionState(model.ConnStateDisconnected)
				fc.appLifecycle.SetAppState(model.AppStateNotConfigured, nil)
				fc.appLifecycle.SetAuthState(model.AuthStateNotAuthenticated)
			}
			if err = fc.configs.SaveToFile(); err != nil {
				log.Error(err)
			}
			if err = fc.states.SaveToFile(); err != nil {
				log.Error(err)
			}
			log.Debugf("App reconfigured . New parameters : %v", fc.configs)

			configReport := model.ConfigReport{
				OpStatus:  opStatus,
				ErrorText: errorText,
				AppState:  *fc.appLifecycle.GetAllStates(),
			}
			msg := fimpgo.NewMessage("evt.app.config_report", model.ServiceName, fimpgo.VTypeObject, configReport, nil, nil, newMsg.Payload)
			msg.Source = "oss"
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}
			fc.states.SaveToFile()

		case "cmd.log.set_level":
			// Configure log level
			level, err := newMsg.Payload.GetStringValue()
			if err != nil {
				return
			}
			logLevel, err := log.ParseLevel(level)
			if err == nil {
				log.SetLevel(logLevel)
				fc.configs.LogLevel = level
				fc.configs.SaveToFile()
			}
			log.Info("Log level updated to = ", logLevel)

		case "cmd.app.factory_reset":
			val := model.ButtonActionResponse{
				Operation:       "cmd.app.factory_reset",
				OperationStatus: "ok",
				Next:            "config",
				ErrorCode:       "",
				ErrorText:       "",
			}
			fc.appLifecycle.SetConfigState(model.ConfigStateNotConfigured)
			fc.appLifecycle.SetAppState(model.AppStateNotConfigured, nil)
			fc.appLifecycle.SetAuthState(model.AuthStateNotAuthenticated)
			msg := fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.thing.get_inclusion_report":
			inclReport := ns.MakeInclusionReport("ossMeter")
			msg := fimpgo.NewMessage("evt.thing.inclusion_report", "oss", "object", inclReport, nil, nil, newMsg.Payload)
			adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "oss", ResourceAddress: "1"}
			fc.mqt.Publish(&adr, msg)

		case "cmd.thing.delete":
			val, err := newMsg.Payload.GetStrMapValue()
			if err != nil {
				log.Error("Wrong msg format")
				return
			}
			deviceID, ok := val["address"]
			if ok {
				val := map[string]interface{}{
					"address": deviceID,
				}
				adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "oss", ResourceAddress: "1"}
				msg := fimpgo.NewMessage("evt.thing.exclusion_report", "oss", fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
				fc.mqt.Publish(adr, msg)
				log.Info("Device with deviceID: ", deviceID, " has been removed from network.")
			} else {
				log.Error("Incorrect address")
			}

		case "cmd.system.reset":
			log.Info("Exluding device")
			val := map[string]interface{}{
				"address": "ossMeter",
			}
			msg := fimpgo.NewMessage("evt.thing.exclusion_report", "oss", fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			msg.Source = "oss"
			adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "oss", ResourceAddress: "1"}
			fc.mqt.Publish(&adr, msg)

			fc.configs.AccessToken = ""
			fc.configs.Email = ""
			fc.configs.EmailCode = ""
			fc.configs.SelectedMeters = ""
			fc.configs.UserID = ""
			fc.configs.SaveToFile()
			fc.states.SaveToFile()
			val2 := model.ButtonActionResponse{
				Operation:       "cmd.system.reset",
				OperationStatus: "ok",
				Next:            "config",
				ErrorCode:       "",
				ErrorText:       "",
			}
			fc.appLifecycle.SetConfigState(model.ConfigStateNotConfigured)
			fc.appLifecycle.SetAppState(model.AppStateNotConfigured, nil)
			fc.appLifecycle.SetAuthState(model.AuthStateNotAuthenticated)
			fc.appLifecycle.SetConnectionState(model.ConnStateDisconnected)
			msg = fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val2, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				log.Error(err)
			}

		case "cmd.app.uninstall":
			log.Info("Exluding device")
			val := map[string]interface{}{
				"address": "ossMeter",
			}
			msg := fimpgo.NewMessage("evt.thing.exclusion_report", "oss", fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			msg.Source = "oss"
			adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "oss", ResourceAddress: "1"}
			fc.mqt.Publish(&adr, msg)
		}
	}
}
