/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"cloudnativeapp/clm/internal"
	"cloudnativeapp/clm/pkg/prober"
	"flag"
	zap1 "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	clmv1beta1 "cloudnativeapp/clm/api/v1beta1"
	"cloudnativeapp/clm/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = clmv1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var logToFile bool
	var logLevel string
	var logFilePath string
	var logFileMaxSize int
	var logFileMaxBackups int
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	// logger related setting
	flag.BoolVar(&logToFile, "enable-log-file", false, "Enable to write log to file.")
	flag.StringVar(&logLevel, "log-level", "info", "The log level. Available: info, debug")
	flag.StringVar(&logFilePath, "log-file-path", "/var/log/clm.log",
		"The path of log if enable-log-file is true.")
	flag.IntVar(&logFileMaxSize, "log-file-maxsize", 200,
		"The maxsize of log file if enable-log-file is true.")
	flag.IntVar(&logFileMaxBackups, "log-file-maxbackups", 3,
		"The max backups of log file if enable-log-file is true.")

	flag.Parse()

	if logToFile {
		w := zapcore.AddSync(&lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    logFileMaxSize,
			MaxBackups: logFileMaxBackups,
		})
		mw := io.MultiWriter(w, os.Stdout)
		// 生产版本的日志格式有点丑陋
		encCfg := zap1.NewDevelopmentEncoderConfig()
		encoder := zapcore.NewConsoleEncoder(encCfg)
		ctrl.SetLogger(zap.New(zap.UseDevMode(parseLogLevel(logLevel)), zap.WriteTo(mw), zap.Encoder(encoder)))
	} else {
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "b554fb75.cloudnativeapp.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.CRDReleaseReconciler{
		Client:  mgr.GetClient(),
		Log:     ctrl.Log.WithName("controllers").WithName("CRDRelease"),
		Scheme:  mgr.GetScheme(),
		Eventer: mgr.GetEventRecorderFor("CRDRelease"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CRDRelease")
		os.Exit(1)
	}
	if err = (&controllers.SourceReconciler{
		Client:  mgr.GetClient(),
		Log:     ctrl.Log.WithName("controllers").WithName("Source"),
		Scheme:  mgr.GetScheme(),
		Eventer: mgr.GetEventRecorderFor("Source"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Source")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	controllers.MGRClient = mgr.GetClient()
	internal.Prober = prober.NewProber()
	controllers.EventRecorder = mgr.GetEventRecorderFor("CRDRelease")
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// Info : only error will print stacktrace, Debug: warn and error will print stacktrace.
func parseLogLevel(level string) bool {
	switch strings.ToLower(level) {
	case "info":
		return false
	case "debug":
		return true
	default:
		setupLog.Info("can not find the log level, set it to 'info' level", "target level", level)
		return false
	}
}
