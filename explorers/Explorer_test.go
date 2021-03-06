package explorer

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type settings struct {
	mx          *sync.RWMutex `yaml:"-"`
	login, pass string        `yaml:"-"`
	bases       []Bases       `yaml:"-"`

	Explorers [] *struct {
		Name     string                 `yaml:"Name"`
		Property map[string]interface{} `yaml:"Property"`
	} `yaml:"Explorers"`

	MSURL  string `yaml:"MSURL"`
	MSUSER string `yaml:"MSUSER"`
	MSPAS  string `yaml:"MSPAS"`
}

type Bases struct {
	Caption  string `json:"Caption"`
	Name     string `json:"Name"`
	UUID     string `json:"UUID"`
	UserName string `json:"UserName"`
	UserPass string `json:"UserPass"`
	Cluster  *struct {
		MainServer string `json:"MainServer"`
		RASServer  string `json:"RASServer"`
		RASPort    int    `json:"RASPort"`
	} `json:"Cluster"`
	URL string `json:"URL"`
}

func (s *settings) GetLogPass(ibname string) (login, pass string){
	for _, base := range s.bases {
		if strings.ToLower(base.Name) == strings.ToLower(ibname) {
			pass = base.UserPass
			login = base.UserName
			break
		}
	}

	return
}

func (s *settings) RAC_Path() string {
	return "/opt/1C/v8.3/x86_64/rac"
}

func (s *settings) GetProperty(explorerName string, propertyName string, defaultValue interface{}) interface{} {
	if v, ok := s.GetExplorers()[explorerName][propertyName]; ok {
		return v
	} else {
		return defaultValue
	}
}

func (s *settings) GetExplorers() map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{}, 0)
	for _, item := range s.Explorers {
		result[item.Name] = item.Property
	}

	return result
}

//////////////////////////////////////////

func Test_Explorer(t *testing.T) {
	for id, test := range initests() {
		t.Run(fmt.Sprintf("Выполняем тест %d", id), test)
	}
}

func initests() []func(*testing.T) {
	s := new(settings)
	if err := yaml.Unmarshal([]byte(settingstext()), s); err != nil {
		panic("Ошибка десириализации настроек")
	}
	metric := new(Metrics).Construct(s)

	siteMux := http.NewServeMux()
	siteMux.Handle("/1C_Metrics", promhttp.Handler())
	siteMux.Handle("/Continue", Continue(metric))
	siteMux.Handle("/Pause", Pause(metric))

	cerror := make(chan error)
	go func() {
		for range cerror {

		}
	}()

	objectlic := new(ExplorerClientLic).Construct(s, cerror)
	objectPerf := new(ExplorerAvailablePerformance).Construct(s, cerror)
	objectMem := new(ExplorerSessionsMemory).Construct(s, cerror)
	objectSes := new(ExplorerSessions).Construct(s, cerror)
	objectCon := new(ExplorerConnects).Construct(s, cerror)
	objectCSJ := new(ExplorerCheckSheduleJob).Construct(s, cerror)
	//objectProc := new(ExplorerProc).Construct(s, cerror)

	metric.Append(objectlic, objectPerf, objectMem, objectSes, objectCon, objectCSJ)

	port := "9999"
	url := "http://localhost:" + port + "/1C_Metrics"
	go http.ListenAndServe(":"+port, siteMux)

	get := func(URL string) (StatusCode int, body string, err error) {
		var resp *http.Response

		if resp, err = http.Get(URL); err != nil {
			return 0, "", fmt.Errorf("Ошибка при обращении к %q:\n %v", url, err)
		}
		defer resp.Body.Close()
		StatusCode = resp.StatusCode

		if body, err := ioutil.ReadAll(resp.Body); err != nil {
			return StatusCode, "", err
		} else {
			return StatusCode, string(body), nil
		}
	}

	return []func(*testing.T){
		func(t *testing.T) {
			t.Parallel()
			StatusCode, _, err := get(url)
			if err != nil {
				t.Errorf("Произошла ошибка %v ", err)
				return
			}
			if StatusCode != 200 {
				t.Error("Код ответа должен быть 200, имеем ", StatusCode)
				return
			}
		},
		func(t *testing.T) {
			// middleware := func(h http.Handler) http.Handler {
			// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 		h.ServeHTTP(w, r)
			// 	})
			// }
			t.Parallel()
			go objectlic.Start(objectlic)
			time.Sleep(time.Second) // Нужно подождать, что бы Explore успел отработаь

			_, body, err := get(url)
			if err != nil {
				t.Error(err)
			} else if strings.Index(body, objectlic.GetName()) < 0 {
				t.Error("В ответе не найден", objectlic.GetName())
			}
		},
		func(t *testing.T) {
			t.Parallel()
			go objectPerf.Start(objectPerf)
			time.Sleep(time.Second) // Нужно подождать, что бы Explore успел отработаь

			_, body, err := get(url)
			if err != nil {
				t.Error(err)
			} else if strings.Index(body, objectPerf.GetName()) < 0 {
				t.Error("В ответе не найден", objectPerf.GetName())
			}
		},
		func(t *testing.T) {
			t.Parallel()
			go objectMem.Start(objectMem)
			time.Sleep(time.Second) // Нужно подождать, что бы Explore успел отработаь

			_, body, err := get(url)
			if err != nil {
				t.Error(err)
			} else if strings.Index(body, objectMem.GetName()) < 0 {
				t.Error("В ответе не найден", objectMem.GetName())
			}
		},
		func(t *testing.T) {
			t.Parallel()
			go objectSes.Start(objectSes)
			time.Sleep(time.Second) // Нужно подождать, что бы Explore успел отработаь

			_, body, err := get(url)
			if err != nil {
				t.Error(err)
			} else if strings.Index(body, objectSes.GetName()) < 0 {
				t.Error("В ответе не найден", objectSes.GetName())
			}
		},
		func(t *testing.T) {
			t.Parallel()
			go objectCon.Start(objectCon)
			time.Sleep(time.Second) // Нужно подождать, что бы Explore успел отработаь

			_, body, err := get(url)
			if err != nil {
				t.Error(err)
			} else if strings.Index(body, objectCon.GetName()) < 0 {
				t.Error("В ответе не найден", objectCon.GetName())
			}
		},
		func(t *testing.T) {
			t.Parallel()
			go objectCSJ.Start(objectCSJ)
			time.Sleep(time.Second) // Нужно подождать, что бы Explore успел отработаь

			_, body, err := get(url)
			if err != nil {
				t.Error(err)
			} else if strings.Index(body, objectCSJ.GetName()) < 0 {
				t.Error("В ответе не найден", objectCSJ.GetName())
			}
		},
		func(t *testing.T) {
			go objectCSJ.Start(objectCSJ)
			go objectCon.Start(objectCon)
			time.Sleep(time.Second) // Нужно подождать, что бы Explore успел отработаь

			get(url)

			code, _, _ := get("http://localhost:" + port + "/Pause?metricNames=SheduleJob,Connect")
			if code != http.StatusOK {
				t.Error("Код ответа должен быть 200, имеем", code)
			}

			_, body, err := get(url)
			if err != nil {
				t.Error(err)
			} else if strings.Index(body, objectCSJ.GetName()) >= 0 || strings.Index(body, objectCon.GetName()) >= 0 {
				t.Error("В ответе найден", objectCSJ.GetName(), "или", objectCon.GetName(), "его там быть не должно")
			}
			// разблокируем
			get("http://localhost:" + port + "/Continue?metricNames=SheduleJob,Connect")
		},
		func(t *testing.T) {
			go objectCSJ.Start(objectCSJ)
			go objectCon.Start(objectCon)
			time.Sleep(time.Second) // Нужно подождать, что бы Explore успел отработаь

			get(url)
			get("http://localhost:" + port + "/Pause?metricNames=SheduleJob,Connect")
			time.Sleep(time.Second)

			code, _, _ := get("http://localhost:" + port + "/Continue?metricNames=SheduleJob,Connect")
			if code != http.StatusOK {
				t.Error("Код ответа должен быть 200, имеем", code)
			}
			time.Sleep(time.Second) // нужно т.к. итерация внутреннего цикла экспортера 1 сек (так в настройках выставлено)
			_, body, err := get(url)
			if err != nil {
				t.Error(err)
			} else if strings.Index(body, objectCSJ.GetName()) < 0 || strings.Index(body, objectCon.GetName()) < 0 {
				t.Error("В ответе не найдены", objectCSJ.GetName(), "или", objectCon.GetName())
			}
		},
		func(t *testing.T) {
			// Нет смысла т.к. эта метрика только под линуксом работает
			//t.Parallel()
			//go objectProc.Start(objectProc)
			//time.Sleep(time.Second*2) // Нужно подождать, что бы Explore успел отработаь
			//
			//_, body, err := get()
			//if err != nil {
			//	t.Error(err)
			//} else if str := body; strings.Index(str, "ProcData") < 0 {
			//	t.Error("В ответе не найден ProcData")
			//}
		},
	}
}

func settingstext() string {
	return `Explorers:
- Name: ClientLic
  Property:
    timerNotyfy: 60
- Name: AvailablePerformance
  Property:
    timerNotyfy: 10
- Name: SheduleJob
  Property:
    timerNotyfy: 1
- Name: Session
  Property:
    timerNotyfy: 60
- Name: Connect
  Property:
    timerNotyfy: 1
- Name: SessionsMemory
  Property:
    timerNotyfy: 10
- Name: ProcData
  Property:
    processes:
      - rphost
      - ragent
      - rmngr
    timerNotyfy: 10
MSURL: http://ca-fr-web-1/fresh/int/sm/hs/PTG_SysExchange/GetDatabase
MSUSER: RemoteAccess
MSPAS: dvt45hn`
}
