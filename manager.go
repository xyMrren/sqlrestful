/*********************************************
                   _ooOoo_
                  o8888888o
                  88" . "88
                  (| -_- |)
                  O\  =  /O
               ____/`---'\____
             .'  \\|     |//  `.
            /  \\|||  :  |||//  \
           /  _||||| -:- |||||-  \
           |   | \\\  -  /// |   |
           | \_|  ''\---/''  |   |
           \  .-\__  `-`  ___/-. /
         ___`. .'  /--.--\  `. . __
      ."" '<  `.___\_<|>_/___.'  >'"".
     | | :  `- \`.;`\ _ /`;.`/ - ` : | |
     \  \ `-.   \_ __\ /__ _/   .-` /  /
======`-.____`-.___\_____/___.-`____.-'======
                   `=---='

^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
           佛祖保佑       永无BUG
           心外无法       法外无心
           三宝弟子       飞猪宏愿
*********************************************/

package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/hashicorp/hcl"
)

// Manager - a macros manager
type Manager struct {
	macros   map[string]*Macro
	compiled *template.Template
	sync.RWMutex
}

func fixMacro(v *Macro) {
	if len(v.Total) > 0 {
		v.Result = "page"
	}

	if v.Result == "" {
		v.Result = "list"
	} else {
		v.Result = strings.ToLower(v.Result)
	}

	if v.Impl == "" {
		v.Impl = "sql"
	} else {
		v.Impl = strings.ToLower(v.Impl)
	}

	if v.Impl != "js" && v.Impl != "cmd" {
		v.Impl = "sql"
	}

	if v.Format == "" {
		v.Format = "enclosed"
	} else {
		v.Format = strings.ToLower(v.Format)
	}

	if v.Format != "origin" && v.Format != "nil" && v.Format != "redirect" {
		v.Format = "enclosed"
	}

	switch {
	case v.Result == "list":
	case v.Result == "object":
	case v.Result == "page":
	default:
		v.Result = "list"
	}

	if v.Summary == "" {
		v.Summary = v.name
	}

	if len(v.Tags) == 0 {
		v.Tags = append(v.Tags, v.name)
	}

	if v.isDefinedSecurity() {
		if v.Security.Policy == "" {
			v.Security.Policy = "include"
		} else {
			v.Security.Policy = strings.ToLower(v.Security.Policy)
		}

		if v.Security.Users != nil && len(v.Security.Users) > 0 {
			v.usersMap = map[string]bool{}
			for _, u := range v.Security.Users {
				v.usersMap[u] = true
			}
		}
		if v.Security.Roles != nil && len(v.Security.Roles) > 0 {
			v.rolesMap = map[string]bool{}
			for _, r := range v.Security.Users {
				v.rolesMap[r] = true
			}
		}
	}

}

// NewManager - initialize a new manager
func NewManager(configpath string) (*Manager, error) {
	manager := new(Manager)
	manager.macros = make(map[string]*Macro)
	manager.compiled = template.New("main")

	for _, p := range strings.Split(configpath, ",") {
		files, _ := filepath.Glob(p)

		if len(files) < 1 {
			return nil, fmt.Errorf("找不到指定的文件路径(%s)！", p)
		}

		for _, file := range files {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				return nil, err
			}

			var config map[string]*Macro
			if err := hcl.Unmarshal(data, &config); err != nil {
				return nil, err
			}

			for k, v := range config {
				manager.macros[k] = v

				if len(v.Exec) > 0 {
					_, err := manager.compiled.New(k).Parse(v.Exec)
					if err != nil {
						return nil, err
					}
				}

				if len(v.Total) > 0 {
					_, err := manager.compiled.New(k + "Total").Parse(v.Total)
					if err != nil {
						return nil, err
					}
				}

				v.methodMacros = make(map[string]*Macro)

				if v.Get != nil {
					v.methodMacros["GET"] = v.Get
				}

				if v.Post != nil {
					v.methodMacros["POST"] = v.Post
				}

				if v.Put != nil {
					v.methodMacros["PUT"] = v.Put
				}

				if v.Patch != nil {
					v.methodMacros["PATCH"] = v.Patch
				}

				if v.Delete != nil {
					v.methodMacros["DELETE"] = v.Delete
				}

				v.manager = manager
				v.name = k

				if v.Path == "" {
					v.Path = "/" + v.name
				}

				if !strings.HasPrefix(v.Path, "/") {
					v.Path = "/" + v.Path
				}

				fixMacro(v)

				for k, childm := range v.methodMacros {
					childm.manager = manager
					childm.Methods = []string{k}
					childm.name = v.name + "." + strings.ToLower(k)
					childm.Path = v.Path
					if childm.Include == nil {
						childm.Include = v.Include
					} else if v.Include != nil {
						childm.Include = append(childm.Include, v.Include[0:]...)
					}
					if childm.Tags == nil {
						childm.Tags = v.Tags
					} else if v.Tags != nil {
						childm.Tags = append(childm.Tags, v.Tags[0:]...)
					}

					if !childm.isDefinedSecurity() {
						childm.Security = v.Security
					}

					fixMacro(childm)
				}

			}
		}
	}

	return manager, nil
}

// Get - fetches the required macro
func (m *Manager) Get(macro string) *Macro {
	m.RLock()
	defer m.RUnlock()

	return m.macros[macro]
}

// Size - return the size of the currently loaded configs
func (m *Manager) Size() int {
	return len(m.macros)
}

// Names - return a list of registered macros
func (m *Manager) Names() (ret []string) {
	for k := range m.macros {
		ret = append(ret, k)
	}

	return ret
}

// List - return all macros
func (m *Manager) List() (ret []*Macro) {
	m.RLock()
	defer m.RUnlock()
	for _, v := range m.macros {
		if !strings.HasPrefix(v.name, "_") {
			ret = append(ret, v)
		}
	}
	return ret
}
