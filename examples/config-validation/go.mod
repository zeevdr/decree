module github.com/zeevdr/decree/examples/config-validation

go 1.24.0

require github.com/zeevdr/decree/sdk/tools v0.3.1

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace github.com/zeevdr/decree/sdk/tools => ../../sdk/tools

replace github.com/zeevdr/decree/api => ../../api

replace github.com/zeevdr/decree/sdk/adminclient => ../../sdk/adminclient
