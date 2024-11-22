package uponor

import (
	"fmt"
	"net/http"

	"github.com/jaanek/hassext/emodul"
	"github.com/jaanek/hassext/httpclient"
	"github.com/zerodha/logf"
)

type ThermostatName string
type EntityState string

const (
	THERMOSTAT_ELUTUBA              ThermostatName = "floor1_elutuba"
	THERMOSTAT_ESIK                 ThermostatName = "floor1_esik"
	THERMOSTAT_DUSSIRUUM            ThermostatName = "floor1_dussiruum"
	THERMOSTAT_SAUNA_EESRUUM        ThermostatName = "floor1_sauna_eesruum"
	THERMOSTAT_ELUTUBA_TARGET       ThermostatName = "floor1_elutuba_target"
	THERMOSTAT_ESIK_TARGET          ThermostatName = "floor1_esik_target"
	THERMOSTAT_DUSSIRUUM_TARGET     ThermostatName = "floor1_dussiruum_target"
	THERMOSTAT_SAUNA_EESRUUM_TARGET ThermostatName = "floor1_sauna_eesruum_target"
	ENTITY_THERMOSTAT_ELUTUBA       EntityState    = "climate.elutuba"
	ENTITY_THERMOSTAT_ESIK          EntityState    = "climate.kook_esik"
	ENTITY_THERMOSTAT_DUSSIRUUM     EntityState    = "climate.dussiruum"
	ENTITY_THERMOSTAT_SAUNA_EESRUUM EntityState    = "climate.sauna_eesruum"
)

var (
	DeviceUponorWallThermostat = emodul.SensorMqttConfigDevice{
		Identifiers:  []string{"hassext_uponor_wall_thermostat"},
		Manufacturer: "UponorC",
		Model:        "Temperature sensor",
		Name:         "Uponor wall thermostat",
	}
	// DeviceFloorWallTempSensorEsik = emodul.SensorMqttConfigDevice{
	// 	Identifiers:  []string{"hassext_" + string(THERMOSTAT_ESIK)},
	// 	Manufacturer: "UponorC",
	// 	Model:        "Temperature sensor",
	// 	Name:         "Floor1 wall temperature sensor esik",
	// }
	// DeviceFloorWallTempSensorDussiruum = emodul.SensorMqttConfigDevice{
	// 	Identifiers:  []string{"hassext_" + string(THERMOSTAT_DUSSIRUUM)},
	// 	Manufacturer: "UponorC",
	// 	Model:        "Temperature sensor",
	// 	Name:         "Floor1 wall temperature sensor dussiruum",
	// }
	// DeviceFloorWallTempSensorSaunaEesruum = emodul.SensorMqttConfigDevice{
	// 	Identifiers:  []string{"hassext_" + string(THERMOSTAT_SAUNA_EESRUUM)},
	// 	Manufacturer: "UponorC",
	// 	Model:        "Temperature sensor",
	// 	Name:         "Floor1 wall temperature sensor sauna eesruum",
	// }
)

type Uponor interface {
	FetchData() error
	GetData() *UponorControllerData
}

type uponor struct {
	lo     logf.Logger
	http   httpclient.HttpClient
	params *HttpClientParams
	data   UponorControllerData
	// stateData data.Data
}

type HttpClientParams struct {
	Host string
}

func NewUponorClient(lo logf.Logger, params *HttpClientParams) Uponor {
	return &uponor{
		lo:     lo,
		http:   httpclient.New(nil, getApiDefaultRetryCheckPolicy(lo, params), defaultRetryWaitDelay, false),
		params: params,
	}
}

func (m *uponor) GetData() *UponorControllerData {
	return &m.data
}

type UponorWaspVar struct {
	VarName  string `json:"waspVarName"`
	VarValue string `json:"waspVarValue"`
}

type UponorControllerData struct {
	ResultCode string `json:"result"`
	Output     struct {
		Vars []UponorWaspVar
	}
}

func (m *uponor) FetchData() error {
	m.data = UponorControllerData{}
	var input = []byte("{}")
	_, err := httpclient.Post(m.http, "http://"+m.params.Host+"/JNAP/", input, func(req *httpclient.Request) {
		req.Header.Set("x-jnap-action", "http://phyn.com/jnap/uponorsky/GetAttributes")
		req.ContentLength = int64(len(input))
	}, func(resp *http.Response) ([]byte, error) {
		body, e := httpclient.ReadJsonResult(resp, &m.data)
		fmt.Println(fmt.Sprintf("[RESPONSE] Status code: %d, status: %s, result: %v", resp.StatusCode, resp.Status, string(body)))
		if e != nil {
			return nil, e
		}
		return body, nil
	})
	if err != nil {
		return fmt.Errorf("http post error: %w", err)
	}
	// m.stateData.Write(result)
	return nil
}

func FahrenheitToCelsius(f float64) float64 {
	// Formula: (°F - 32) × 5/9 = °C
	return (f - 32) * 5 / 9
}
