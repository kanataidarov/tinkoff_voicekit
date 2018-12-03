package main

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/kanataidarov/tinkoff_voicekit/pkg/args"
	"github.com/kanataidarov/tinkoff_voicekit/pkg/common"
	sttPb "github.com/kanataidarov/tinkoff_voicekit/pkg/tinkoff/cloud/stt/v1"
)

func main() {
	opts := args.ParseStreamingRecognizeOptions()
	if opts == nil {
		os.Exit(1)
	}
	defer func(InputFile *os.File) {
		_ = InputFile.Close()
	}(opts.InputFile)

	var dataReader io.Reader
	if strings.HasSuffix(opts.InputFile.Name(), ".wav") {
		reader, err := common.OpenWavFormat(opts.InputFile, *opts.Encoding, *opts.NumChannels, *opts.Rate)
		if err != nil {
			panic(err)
		}
		dataReader = reader
	} else {
		dataReader = opts.InputFile
	}

	client, err := common.NewSttClient(opts.CommonOptions)
	if err != nil {
		panic(err)
	}
	defer func(client common.SpeechToTextClient) {
		_ = client.Close()
	}(client)

	// NOTE: in production code you should probably use context.WithCancel, context.WithDeadline or context.WithTimeout
	// instead of context.Background()
	stream, err := client.StreamingRecognize(context.Background())
	if err != nil {
		panic(err)
	}
	reqeust := &sttPb.StreamingRecognizeRequest{
		StreamingRequest: &sttPb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &sttPb.StreamingRecognitionConfig{
				Config: &sttPb.RecognitionConfig{
					Encoding:                   sttPb.AudioEncoding(sttPb.AudioEncoding_value[*opts.Encoding]),
					SampleRateHertz:            uint32(*opts.Rate),
					LanguageCode:               *opts.LanguageCode,
					MaxAlternatives:            uint32(*opts.MaxAlternatives),
					ProfanityFilter:            !(*opts.DisableProfanityFilter),
					EnableAutomaticPunctuation: !(*opts.DisableAutomaticPunctuation),
					NumChannels:                uint32(*opts.NumChannels),
				},
				SingleUtterance: *opts.SingleUtterance,
				InterimResultsConfig: &sttPb.InterimResultsConfig{
					EnableInterimResults: *opts.InterimResults,
				},
			},
		},
	}
	if *opts.DoNotPerformVad {
		reqeust.GetStreamingConfig().Config.Vad = &sttPb.RecognitionConfig_DoNotPerformVad{DoNotPerformVad: true}
	} else {
		reqeust.GetStreamingConfig().Config.Vad = &sttPb.RecognitionConfig_VadConfig{
			VadConfig: &sttPb.VoiceActivityDetectionConfig{
				SilenceDurationThreshold: float32(*opts.SilenceDurationThreshold),
			},
		}
	}

	if err := stream.Send(reqeust); err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func(stream sttPb.SpeechToText_StreamingRecognizeClient) {
			_ = stream.CloseSend()
		}(stream)
		for {
			buffer := make([]byte, 1024)
			bytesRead, err := dataReader.Read(buffer)

			if err == io.EOF {
				break
			}
			if err != nil {
				panic(err)
			}

			err = stream.Send(&sttPb.StreamingRecognizeRequest{
				StreamingRequest: &sttPb.StreamingRecognizeRequest_AudioContent{
					AudioContent: buffer[:bytesRead],
				},
			})
			if err != nil {
				panic(err)
			}
		}

	}()

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		if common.PrettyPrintProtobuf(msg) != nil {
			panic(err)
		}
	}

	wg.Wait()

}