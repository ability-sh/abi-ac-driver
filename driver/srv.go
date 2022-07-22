package driver

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-lib/errors"
	"github.com/ability-sh/abi-lib/json"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/ability-sh/abi-micro/runtime"
	unit "unit.nginx.org/go"
)

func Run(executor Executor) error {

	AC_APPID := os.Getenv("AC_APPID")
	AC_VER := os.Getenv("AC_VER")
	AC_ABILITY := os.Getenv("AC_ABILITY")

	AC_ENV := os.Getenv("AC_ENV")
	AC_ADDR := os.Getenv("AC_ADDR")
	AC_CONFIG := os.Getenv("AC_CONFIG")
	AC_HTTP_BODY_SIZE, _ := strconv.ParseInt(os.Getenv("AC_HTTP_BODY_SIZE"), 10, 64)

	if AC_HTTP_BODY_SIZE == 0 {
		AC_HTTP_BODY_SIZE = 1024 * 1024 * 500
	}

	var config interface{} = nil
	var err error = nil

	if AC_ENV == "unit" {

		err = json.Unmarshal([]byte(AC_CONFIG), &config)

		if err != nil {
			return err
		}

	} else {

		config, err = GetConfig("./config.yaml")

		if err != nil {
			return err
		}
	}

	p := runtime.NewPayload()

	err = p.SetConfig(config)

	if err != nil {
		return err
	}

	defer p.Exit()

	info, _ := GetAppInfo()

	alias := dynamic.StringValue(dynamic.Get(config, "alias"), "/")

	if !strings.HasSuffix(alias, "/") {
		alias = alias + "/"
	}

	alias_n := len(alias)

	s_state := alias + "__stat"

	http.HandleFunc(alias, func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == s_state {
			setDataResponse(w, map[string]interface{}{"appid": AC_APPID, "ver": AC_VER, "ability": AC_ABILITY, "env": AC_ENV})
			return
		}

		if strings.HasSuffix(r.URL.Path, ".json") {

			var name = r.URL.Path[alias_n:]

			trace := r.Header.Get("Trace")

			if trace == "" {
				r.Header.Get("trace")
			}

			if trace == "" {
				trace = micro.NewTrace()
				w.Header().Add("Trace", trace)
			}

			dynamic.Each(dynamic.Get(info, "cors"), func(key interface{}, value interface{}) bool {
				w.Header().Add(dynamic.StringValue(key, ""), dynamic.StringValue(value, ""))
				return true
			})

			ctx, err := p.NewContext(name, trace)

			if err != nil {
				setErrorResponse(w, err)
				return
			}

			defer ctx.Recycle()

			clientIp := getClientIp(r)

			ctx.SetValue("clientIp", clientIp)

			ctx.AddTag("clientIp", clientIp)

			var inputData interface{} = nil
			ctype := r.Header.Get("Content-Type")

			if ctype == "" {
				ctype = r.Header.Get("content-type")
			}

			if strings.Contains(ctype, "multipart/form-data") {
				inputData = map[string]interface{}{}
				r.ParseMultipartForm(AC_HTTP_BODY_SIZE)
				if r.MultipartForm != nil {
					for key, values := range r.MultipartForm.Value {
						dynamic.Set(inputData, key, values[0])
					}
					for key, values := range r.MultipartForm.File {
						dynamic.Set(inputData, key, values[0])
					}
				}
			} else if strings.Contains(ctype, "json") {

				b, err := ioutil.ReadAll(r.Body)
				defer r.Body.Close()

				if err == nil {
					json.Unmarshal(b, &inputData)
				}

			} else {

				inputData = map[string]interface{}{}

				r.ParseForm()

				for key, values := range r.Form {
					dynamic.Set(inputData, key, values[0])
				}

			}

			rs, err := executor.Exec(ctx, name, inputData)

			if err != nil {
				setErrorResponse(w, err)
				return
			}

			setDataResponse(w, rs)

			return
		}

		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
	})

	if AC_ADDR == "" {
		AC_ADDR = ":8084"
	}

	if AC_ENV == "unit" {
		return unit.ListenAndServe(AC_ADDR, nil)
	} else {
		log.Println("HTTPD", AC_ADDR)
		return http.ListenAndServe(AC_ADDR, nil)
	}

}

func setErrorResponse(w http.ResponseWriter, err error) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	e, ok := err.(*errors.Error)
	if ok {
		b, _ := json.Marshal(e)
		w.Write(b)
	} else {
		b, _ := json.Marshal(map[string]interface{}{"errno": 500, "errmsg": err.Error()})
		w.Write(b)
	}
}

func setDataResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	b, _ := json.Marshal(map[string]interface{}{"errno": 200, "data": data})
	w.Write(b)
}

var clientKeys = []string{"X-Forwarded-For", "x-forwarded-for"}

func getClientIp(r *http.Request) string {

	for _, key := range clientKeys {
		v := r.Header.Get(key)
		if v != "" {
			return strings.Split(v, ",")[0]
		}
	}

	return strings.Split(r.RemoteAddr, ":")[0]
}
