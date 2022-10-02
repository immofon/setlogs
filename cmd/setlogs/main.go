package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/immofon/setlogs"
	"github.com/urfave/cli/v2"
)

func Errorln(a ...any) {
	fmt.Println(a...)
	os.Exit(1)
}

func Errorf(format string, a ...any) {
	fmt.Printf(format, a...)
	os.Exit(1)
}

var userhomedir, userhomedir_err = os.UserHomeDir()

var RootPath = userhomedir + "/.setlogs"
var ConfigFileName = "config.json"

func ConfigFilePath() string {
	return RootPath + "/" + ConfigFileName
}

type Base struct {
	Name      string
	Comment   string
	NextLogID int
}

func InitBase(name string) Base {
	return Base{
		Name:      name,
		Comment:   "",
		NextLogID: 0,
	}
}

func (base *Base) NextLogName(log_type setlogs.Type) string {
	name := fmt.Sprintf("%s/%s/%08d_%s.json", RootPath, base.Name, base.NextLogID, log_type)
	base.NextLogID += 1
	return name
}

func (base Base) Load() setlogs.SetLog {
	path := fmt.Sprintf("%s/%s", RootPath, base.Name)
	dir, err := os.Open(path)
	if err != nil {
		Errorf("read setlogs for base named %q: %v\n", base.Name, err)
	}

	dir_entrys, err := dir.ReadDir(-1)
	if err != nil {
		Errorf("read setlogs for base named %q: %v\n", base.Name, err)
	}

	log_names := make([]string, 0)
	for _, entry := range dir_entrys {
		if strings.HasSuffix(entry.Name(), ".json") && entry.IsDir() == false {
			log_names = append(log_names, entry.Name())
		}
	}

	sort.Sort(sort.StringSlice(log_names))

	logs := setlogs.New(setlogs.TypeEmpty)

	for _, log_name := range log_names {
		f, err := os.Open(path + "/" + log_name)
		if err != nil {
			Errorf("load %s/%s: %v\n", base.Name, log_name, err)
		}
		logs.Merge(setlogs.ReadJSON(f))
		f.Close()
	}
	return logs
}

func Load(base_name string) (Config, Base) {
	name := strings.TrimSpace(base_name)
	config := MustReadConfig()
	base, ok := config.Bases[name]
	if !ok {
		Errorf("Base named %q does not exist!\n", name)
	}
	return config, base
}

func SaveSetLog(base_name string, logs setlogs.SetLog) error {
	base_name = strings.TrimSpace(base_name)
	config, base := Load(base_name)
	err := SafeWriteFileJSON(base.NextLogName(logs.Type), logs)
	if err != nil {
		return err
	}

	config.Bases[base_name] = base
	return config.Save()
}

type Config struct {
	Bases map[string]Base // key: Base.Name
}

func (config Config) Save() error {
	return SafeWriteFileJSON(ConfigFilePath(), config)
}

func ReadConfig() (Config, error) {
	var config Config

	f, err := os.Open(ConfigFilePath())
	if err != nil {
		return config, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&config)
	return config, err
}

func MustReadConfig() Config {
	config, err := ReadConfig()
	if err != nil {
		Errorln("read config.json:", err)
	}
	return config

}

func SafeWriteFile(filename string, data []byte) error {
	tmpfilename := filename + ".safe_tmp"
	f, err := os.OpenFile(tmpfilename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0400)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(data)
	err = f.Sync()
	if err != nil {
		return err
	}

	return os.Rename(tmpfilename, filename)
}

func SafeWriteFileJSON(filename string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return SafeWriteFile(filename, data)
}

func Mkdir(name string) error {
	return os.Mkdir(name, 0700)
}

func main() {
	app := &cli.App{
		Name:  "setlogs",
		Flags: []cli.Flag{},
		Commands: []*cli.Command{
			{
				Name:        "init",
				Aliases:     []string{},
				Flags:       []cli.Flag{},
				Usage:       "Use this command if you didn't use it",
				Description: "init setlogs",
				Action: func(c *cli.Context) error {
					fileinfo, err := os.Stat(RootPath)
					if err == nil {
						if !fileinfo.IsDir() {
							Errorf("%s exists but it is not dir! You should delete this file if you want to use this program!", RootPath)
						} else {
							Errorf("%s exists! You should delete this file if you want to remove all your bases!", RootPath)
						}
					}

					err = Mkdir(RootPath)
					if err != nil {
						Errorf("create dir: %s; error: %v\n", RootPath, err)
					}

					// init config
					err = Config{
						Bases: make(map[string]Base),
					}.Save()
					if err != nil {
						Errorln("init config.json", err)
					}

					return nil
				},
			},
			{
				Name:    "csv",
				Aliases: []string{},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "file", Value: "", Usage: "MUST SET! csv filename"},
					&cli.StringFlag{Name: "name", Value: "", Usage: "MUST SET! base name"},
					&cli.StringFlag{Name: "type", Value: "base", Usage: "the type of setlogs, must be base|mutate|set"},
				},
				Usage:       "init base by read .csv file",
				Description: "Read.csv file to init base",
				Action: func(c *cli.Context) error {
					name := strings.TrimSpace(c.String("name"))
					setlogs_type := setlogs.Type(strings.TrimSpace(c.String("type")))

					if name == "" {
						Errorln("You MUST set --name")
					}
					if setlogs_type != setlogs.TypeBase && setlogs_type != setlogs.TypeMutate && setlogs_type != setlogs.TypeSet {
						Errorln("You MUST set --type as base|mutate|set")
					}

					config := MustReadConfig()

					f, err := os.Open(c.String("file"))
					if err != nil {
						Errorln("open csv file", err)
					}
					defer f.Close()

					logs := setlogs.ReadCSV(f, setlogs_type)

					base, ok := config.Bases[name]
					if setlogs_type == setlogs.TypeBase {
						if ok {
							Errorf("base named %q exists, please choose another name!\n", name)
						}
						err = Mkdir(RootPath + "/" + name)
						if err != nil {
							Errorln("mkdir", err)
						}
						base = InitBase(name)
					} else {
						if !ok {
							Errorf("You must init base named %q first!\n", name)
						}
					}

					err = SafeWriteFileJSON(base.NextLogName(setlogs_type), logs)
					if err != nil {
						Errorln("save log", err)
					}

					config.Bases[name] = base
					err = config.Save()
					if err != nil {
						Errorln("save config.json", err)
					}

					return nil
				},
			},
			{
				Name:    "view",
				Aliases: []string{},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Value: "", Usage: "MUST SET! base name"},
				},
				Usage:       "Use this command if you didn't use it",
				Description: "view base named --name",
				Action: func(c *cli.Context) error {
					name := strings.TrimSpace(c.String("name"))
					_, base := Load(name)
					logs := base.Load()

					logs.TableFprintln(os.Stdout)
					return nil
				},
			},
			{
				Name:    "plugin",
				Aliases: []string{},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "base", Value: "", Usage: "MUST SET! base name"},
				},
				Usage:       "Use this command if you didn't use it",
				Description: "run plugin",
				Action: func(c *cli.Context) error {
					args := c.Args().Slice()
					if len(args) < 1 {
						Errorln("setlogs plugin [options] <plugin_name> [plugin args]")
					}
					plugin_name := args[0]
					args = args[1:]
					plugin_error_must_be_not_nil := func(err error) {
						if err != nil {
							Errorf("Plugin error[%s]:%v\n", plugin_name, err)
						}
					}

					_, base := Load(c.String("base"))
					logs := base.Load()

					switch plugin_name {
					case "jlu-mathlab-homework-count":
						if len(args) == 0 {
							Errorln("Plugin usage: jlu-mathlab-homework-count <homework-path> [key]")
						}

						homework_dir_path := args[0]
						key := ""
						value := "T"
						view_only := true
						if len(args) == 1 {
						} else if len(args) == 2 {
							view_only = false
							key = strings.TrimSpace(args[1])
							if len(key) == 0 {
								Errorln("Plugin usage: You must set key!")
							}

						} else {
							Errorln("Plugin usage: jlu-mathlab-homework-count <homework-path> [key value commit-message]")
						}
						if view_only {
							key = "@new"
						}

						commit_message := fmt.Sprintf("Set Record[%q] = %q which satisfied condition. generated by plugin[%s] from homework-path: %s", key, value, plugin_name, homework_dir_path)

						homework_dir, err := os.Open(homework_dir_path)
						plugin_error_must_be_not_nil(err)

						homework_dir_entrys, err := homework_dir.ReadDir(-1)
						plugin_error_must_be_not_nil(err)

						stdents_id_set := setlogs.NewSet()

						numbers_re := regexp.MustCompile("[0-9]+")
						for _, homework_dir_entry := range homework_dir_entrys {
							student_id := numbers_re.FindString(homework_dir_entry.Name())
							if len(student_id) > 0 {
								stdents_id_set[student_id] = true
							}
						}

						mutates := setlogs.New(setlogs.TypeMutate)
						mutates.Comment = commit_message
						for student_id := range stdents_id_set {
							mutates.AppendRecords(setlogs.Record{
								setlogs.ID: student_id,
								key:        value,
							})
						}

						mutates.TableFprintln(os.Stdout)

						logs.Merge(mutates)

						if view_only {
							logs.TableFprintln(os.Stdout)
							break
						} else {
							SaveSetLog(base.Name, mutates)
							break
						}
					default:
						Errorln("Not found plugin:", plugin_name)
					}

					return nil
				},
			},
		},
	}
	_ = app.Run(os.Args)
}
