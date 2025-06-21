package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "vn",
	Short: "VN - Vulnerability Navigator: A CLI tool for OWASP Top 10 security testing",
	Long: color.New(color.FgCyan).Sprint(`
██╗   ██╗███╗   ██╗
██║   ██║████╗  ██║
██║   ██║██╔██╗ ██║
╚██╗ ██╔╝██║╚██╗██║
 ╚████╔╝ ██║ ╚████║
  ╚═══╝  ╚═╝  ╚═══╝

VN - Vulnerability Navigator
A powerful CLI tool for security testing based on OWASP Top 10.

Currently supports:
• SQL Injection Testing
• More vulnerabilities coming soon...`),
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func GetRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.vn.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringP("output", "o", "console", "output format (console, json, xml)")
	
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".vn")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
} 