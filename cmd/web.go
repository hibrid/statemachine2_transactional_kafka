// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hibrid/statemachine2_transactional_kafka/lib/app"
	"github.com/hibrid/statemachine2_transactional_kafka/lib/web"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// webCmd represents the web command
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		rawRoot := chi.NewRouter()

		apiRouter := chi.NewRouter().With(
			middleware.Compress(5, "gzip"),
			middleware.NoCache,
			cors.New(cors.Options{
				Debug:            false,
				AllowCredentials: true,
				AllowedOrigins:   []string{"*"}, // set this to us, wild card for this code challenge
				AllowedHeaders:   []string{"Origin", "Accept", "Content-Type", "Authorization"},
				AllowedMethods:   []string{"POST", "GET"},
			}).Handler,
		)
		//show me requests that take longer than configured
		apiRouter.Use(web.NoSessionLogger(logRequestsAfter))
		var root chi.Router = chi.NewRouter()

		rawRoot.Mount("/status", apiRouter)
		rawRoot.Mount("/", root)
		root.NotFound(web.NoExtOrNotFound(root.ServeHTTP))

		httpListener, err := newListener(app.SiteHTTPPort)
		if err != nil {
			log.Fatal("Failed to create http listener", zap.Error(err))
		}

		//not adding TLS support yet, but we'd have another entry for the endpoints to be secured
		//and this entry would map to the routes that we're ok not being secured. like the docs
		spawnOrDie("http", httpListener, func() error { return http.Serve(httpListener, rawRoot) })

		log.Info("Started server", zap.Duration("time", time.Now().Sub(startTime)))

		//again, later would pass something like httpsListener
		waitForShutdown(httpListener)
		log.Info("Done")
	},
}

func newListener(address string) (net.Listener, error) {
	return net.Listen("tcp", address)
}

func init() {
	rootCmd.AddCommand(webCmd)

	webCmd.Flags().StringVar(&app.SiteHTTPPort, "HTTP_LISTEN", ":8080", "port to listen on")
	webCmd.Flags().DurationVar(&logRequestsAfter, "LOG_REQUEST_AFTER", logRequestsAfter, "log requests that take longer than this")

}
