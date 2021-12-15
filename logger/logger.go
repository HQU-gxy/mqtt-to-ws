package logger

import "go.uber.org/zap"

var L, _ = zap.NewDevelopment()
var Lsugar = L.Sugar()
