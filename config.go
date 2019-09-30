// Package config toml语法配置组件
/*关于配置:
1.各组件功能的配置往往有三种方式:
  a.默认值;
  b.配置文件;
  c.使用者代码直接设置
2.因为配置文件是最灵活的,按理三个配置优先级b>c>a这样应该是比较理想的.
3.但因为使用者代码设置较难控制,需要每个组件都仔细封装设置的代码,避免代码设置覆盖配置文件,有时甚至是不可能的任务.
4.所以gobase约定/实现的优先级顺序是: c>b>a, 也就是说一旦有代码设置,配置文件就无效了.
5.建议:能用配置文件控制的,使用者不要用代码再设置.
*/
package config

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	config_duration "github.com/unlikezy/config/duration"

	"github.com/spf13/pflag"
	"github.com/unlikezy/go-defaults"

	"github.com/BurntSushi/toml"
	"github.com/juju/errors"
)

/*
为了各模块能在init时就读取到配置,所以专门引入了pflag(他支持忽略不认识的flag),声明一个独立的FlagSet config_commandline,用于解析confpath
为了让默认的flag和pflag不抱怨confpath没定义,同时也是为了在help时能打印出confpath说明,这里也分别给pflag和flag注册了confpath
*/
var config_commandline = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
var ConfPath = config_commandline.String("confpath", "../conf/config.toml", "config file path. MUST use --confpath style")
var PrintConf = config_commandline.Bool("printconf", true, "printconf . MUST use --printconf style")
var _ = pflag.String("confpath", "../conf/config.toml", "config file path. MUST use --confpath style")
var _ = pflag.Bool("printconf", true, "printconf . MUST use --printconf style")
var _ = flag.String("confpath", "../conf/config.toml", "config file path. MUST use --confpath style")
var _ = flag.Bool("printconf", true, "printconf . MUST use --printconf style")

//仅在单元测试中,允许直接通过函数设置配置文件路径
func SetConfPathForTest(path string) {
	*ConfPath = path
}

func init() {
	//config_commandline仅尽力解析confpath
	config_commandline.Usage = func() {}
	config_commandline.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{
		UnknownFlags: true, //忽略不认识的flag
	}

	config_commandline.Parse(os.Args[1:])
}

const (
	LogLevelNull    = 0
	LogLevelTrace   = 1
	LogLevelDebug   = 2
	LogLevelInfo    = 3
	LogLevelWarning = 4
	LogLevelError   = 5
	LogLevelFatal   = 6
)

var logLevelMap = map[string]uint8{
	"trace": LogLevelTrace,
	"debug": LogLevelDebug,
	"info":  LogLevelInfo,
	"warn":  LogLevelWarning,
	"error": LogLevelError,
	"fatal": LogLevelFatal,
}
var logLevelStrMap = map[uint8]string{
	LogLevelTrace:   "trace",
	LogLevelDebug:   "debug",
	LogLevelInfo:    "info",
	LogLevelWarning: "warn",
	LogLevelError:   "error",
	LogLevelFatal:   "fatal",
}

// LogLevel 日志级别
type LogLevel uint8

// UnmarshalText 通过字符串解析日志级别
func (l *LogLevel) UnmarshalText(text []byte) error {
	level, ok := logLevelMap[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("not support log level %v", string(text))
	}
	*l = LogLevel(level)
	return nil
}

// String 日志级别字符串展示
func (l LogLevel) String() string {
	name, ok := logLevelStrMap[uint8(l)]
	if ok {
		return name
	}

	return "unknown"
}

// Level return uint8 level
func (l LogLevel) Level() uint8 {
	return uint8(l)
}

// Value return uint8 level
func (l LogLevel) Value() uint8 {
	return uint8(l)
}

// LogSize 日志文件大小 B K M G
type LogSize int64

// UnmarshalText 通过字符串解析日志大小
func (l *LogSize) UnmarshalText(text []byte) error {
	if len(text) < 2 { //至少两个字节
		return fmt.Errorf("not support log size %v", string(text))
	}
	c := strings.ToLower(string(text[len(text)-1:])) //最后一个字符
	n, e := strconv.ParseInt(string(text[:len(text)-1]), 10, 64)
	if e != nil {
		return e
	}
	if c == "k" {
		n *= 1024
	} else if c == "m" {
		n *= 1024 * 1024
	} else if c == "g" {
		n *= 1024 * 1024 * 1024
	}
	*l = LogSize(n)
	return nil
}

// String 日志大小字符串展示
func (l LogSize) String() string {
	if l < 1024 {
		return fmt.Sprintf("%dB", int64(l))
	} else if l < 1024*1024 {
		return fmt.Sprintf("%dK", int64(l)/1024)
	} else if l < 1024*1024*1024 {
		return fmt.Sprintf("%dM", int64(l)/1024/1024)
	} else if l < 1024*1024*1024*1024 {
		return fmt.Sprintf("%dG", int64(l)/1024/1024/1024)
	}

	return "unknown"
}

// Size return int64 size
func (l LogSize) Size() int64 {
	return int64(l)
}

// Value return int64 size
func (l LogSize) Value() int64 {
	return int64(l)
}

type Duration = config_duration.Duration

var printFilepathOnce sync.Once

// Parse parse config with default and config file ../conf/config.toml
func Parse(c interface{}) error {
	defaults.SetDefaults(c)
	printFilepathOnce.Do(func() {
		if *PrintConf {
			fmt.Printf("\nconfig file:%s", *ConfPath)
		}
	})
	err := ParseConfigWithoutDefaults(c)
	if err != nil {
		return err
	}
	if *PrintConf {
		fmt.Printf("\n%s", SprintToml(c))
	}
	return nil
}

// ParseConfig same as Parse
func ParseConfig(c interface{}) error {
	return Parse(c)
}

// ParseConfigWithPath 自己定义配置文件路径
func ParseConfigWithPath(c interface{}, path string) error {
	defaults.SetDefaults(c)
	if *PrintConf {
		fmt.Printf("\nconfig file:%s", path)
	}
	if err := DecodeWithEnv(path, c); err != nil {
		fmt.Println(err)
		return err
	}
	if *PrintConf {
		fmt.Printf("\n%s", SprintToml(c))
	}
	return nil
}

// ParseConfigWithoutDefaults no default value
func ParseConfigWithoutDefaults(c interface{}) error {
	if err := DecodeWithEnv(*ConfPath, c); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func DecodeWithEnv(fpath string, c interface{}) error {
	cf, err := ioutil.ReadFile(fpath)
	if err != nil {
		return errors.Annotatef(err, "DecodeWithEnv->ReadFile")
	}

	/* template solution
	ct := template.Must(template.New("ct").Parse(string(cf)))

	envMap := make(map[string]string)
	for _, v := range os.Environ() {
		split_v := strings.Split(v, "=")
		if len(split_v) == 2 {
			envMap[split_v[0]] = split_v[1]
		}
	}

	buf := new(bytes.Buffer)
	err = ct.Execute(buf, envMap)
	if err != nil {
		return errors.Annotatef(err, "DecodeWithEnv->Execute")
	}

	_, err = toml.Decode(buf.String(), c)
	*/

	//os.Expand solution
	mapper := func(escapeDollar string) string {
		if escapeDollar == "$" {
			return "$"
		} else {
			return os.Getenv(escapeDollar)
		}
	}
	confStr := os.Expand(string(cf), mapper)
	_, err = toml.Decode(confStr, c)
	if err != nil {
		return errors.Annotatef(err, "DecodeWithEnv->Decode")
	}
	return nil
}

func SprintToml(c interface{}) string {
	var tmp bytes.Buffer
	toml.NewEncoder(&tmp).Encode(c)
	return tmp.String()
}
