package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/erdaltsksn/cui"
	"github.com/konradit/bereal-timelapse/pkg/bereal"
	"github.com/spf13/cobra"
)

func reverse(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render timelapse",
	Run: func(cmd *cobra.Command, args []string) {
		phone_number, err := cmd.Flags().GetString("phone_number")
		if err != nil {
			cui.Error("Problem parsing phone_number", err)
			os.Exit(1)
		}
		fps, err := cmd.Flags().GetInt("fps")
		if err != nil {
			cui.Error("Problem parsing frames per second (fps)", err)
			os.Exit(1)
		}
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			cui.Error("Problem parsing output", err)
			os.Exit(1)
		}

		if phone_number == "" {
			cui.Error("Phone number needs to be set")
			os.Exit(1)
		}

		b := bereal.BeReal{
			Debug: true,
		}

		err = b.SendAuthMessage(phone_number)
		if err != nil {
			cui.Error(err.Error())
			os.Exit(1)
		}
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Enter SMS 2FA code: ")
		text, _ := reader.ReadString('\n')
		code := strings.Replace(text, "\n", "", -1)
		code = strings.Replace(code, "\r", "", -1)
		err = b.VerifyAuthMessage(code)
		if err != nil {
			cui.Error(err.Error())
			os.Exit(1)
		}
		memories, err := b.GetMemories()
		if err != nil {
			cui.Error(err.Error())
			os.Exit(1)
		}

		// make timelapse

		reverse(memories)
		for index, memory := range memories {
			err = bereal.DownloadFile(fmt.Sprintf("output/front/memory_%d.jpg", index), memory.Secondary.URL)
			if err != nil {
				cui.Error(err.Error())
				os.Exit(1)
			}
			err = bereal.DownloadFile(fmt.Sprintf("output/back/memory_%d.jpg", index), memory.Primary.URL)
			if err != nil {
				cui.Error(err.Error())
				os.Exit(1)
			}

			err = Superimpose(fmt.Sprintf("output/back/memory_%d.jpg", index), fmt.Sprintf("output/front/memory_%d.jpg", index), fmt.Sprintf("output/render_%d.jpg", index))
			if err != nil {
				cui.Error(err.Error())
				os.Exit(1)
			}
		}

		ffmpegArgs := "-y -f image2 -framerate " + fmt.Sprint(fps) + " -i output/render_%d.jpg " + output
		ffmpegCmd := exec.Command("ffmpeg", strings.Split(ffmpegArgs, " ")...)

		stderr, _ := ffmpegCmd.StderrPipe()
		ffmpegCmd.Start()

		scanner := bufio.NewScanner(stderr)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			m := scanner.Text()
			fmt.Println(m)
		}
		ffmpegCmd.Wait()

	},
}

func init() {
	rootCmd.AddCommand(renderCmd)

	renderCmd.Flags().StringP("phone_number", "p", "", "Phone Number: +XXYYYYYYYYY")
	renderCmd.Flags().IntP("fps", "f", 5, "Frames per second")
	renderCmd.Flags().StringP("output", "o", "render.mp4", "Output filename")
	renderCmd.MarkFlagRequired("phone_number")
}
