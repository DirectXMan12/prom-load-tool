package main

import (
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	promdto "github.com/prometheus/client_model/go"
)

func randLetter(r *rand.Rand) rune {
	return rune(r.Intn('z'-'a'+1) + 'a')
}

func randString(r *rand.Rand) string {
	targetLen := r.Intn(31) + 1
	res := make([]rune, targetLen)
	for i := range res {
		res[i] = randLetter(r)
	}

	return string(res)
}

func randomSeries(numSeries int, r *rand.Rand) []*promdto.Metric {
	numFixedLabelValues := 2
	fixedKey := "fixed_label"

	res := make([]*promdto.Metric, numSeries)
	for i := range res {
		metric := &promdto.Metric{
			Gauge: &promdto.Gauge{},
		}
		for lblInd := 0; lblInd < r.Intn(9)+1; lblInd++ {
			k := randString(r)
			v := randString(r)
			metric.Label = append(metric.Label, &promdto.LabelPair{
				Name:  &k,
				Value: &v,
			})
		}
		fixedValue := string([]rune{'a' + rune(r.Intn(numFixedLabelValues))})
		metric.Label = append(metric.Label, &promdto.LabelPair{
			Name:  &fixedKey,
			Value: &fixedValue,
		})
		res[i] = metric
	}

	return res
}

func randomFamilies(numFamilies, maxSeriesPerFamily int, r *rand.Rand) []*promdto.MetricFamily {
	res := make([]*promdto.MetricFamily, numFamilies)
	for i := range res {
		numMetricsInFamily := r.Intn(maxSeriesPerFamily/2) + maxSeriesPerFamily/2
		typ := promdto.MetricType_GAUGE
		metricName := randString(r)
		family := &promdto.MetricFamily{
			Name:   &metricName,
			Type:   &typ,
			Metric: randomSeries(numMetricsInFamily, r),
		}

		res[i] = family
	}

	return res
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s NUM_RANDOM_FAMILIES MAX_SERIES_PER_FAMILY\n", os.Args[0])
		os.Exit(1)
	}

	numFamilies, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Printf("usage: %s NUM_RANDOM_FAMILIES MAX_SERIES_PER_FAMILY\n", os.Args[0])
		log.Fatalf("error: NUM_RANDOM_FAMILIES must be an integer: %v", err)
		os.Exit(2)
	}

	maxSeriesPerFamily, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Printf("usage: %s NUM_RANDOM_METRICS MAX_SERIES_PER_FAMILY\n", os.Args[0])
		log.Fatalf("error: MAX_SERIES_PER_FAMILY must be an integer: %v", err)
		os.Exit(2)
	}

	startTime := time.Now()
	r := rand.New(rand.NewSource(0))
	allSeriesInfo := randomFamilies(numFamilies, maxSeriesPerFamily, r)
	log.Printf("Done generating random series: %s", time.Now().Sub(startTime))

	for _, family := range allSeriesInfo {
		log.Printf("- %s: %v series", *family.Name, len(family.Metric))
	}

	mu := &sync.Mutex{}

	var gatherer prometheus.GathererFunc = func() ([]*promdto.MetricFamily, error) {
		mu.Lock()
		defer mu.Unlock()
		start := time.Now()
		for _, family := range allSeriesInfo {
			for _, metric := range family.Metric {
				val := r.NormFloat64()
				metric.Gauge.Value = &val
			}
		}
		log.Printf("generated new metric values: %v", time.Now().Sub(start))

		return allSeriesInfo, nil
	}

	go func() {
		for {
			time.Sleep(15 * time.Second)
			mu.Lock()
			start := time.Now()
			for _, family := range allSeriesInfo {
				numToReplace := r.Intn(5*len(family.Metric)/6) + len(family.Metric)/6
				newMetrics := randomSeries(numToReplace, r)
				for i, series := range newMetrics {
					family.Metric[i] = series
				}
			}
			log.Printf("replaced some old series with new ones: %v", time.Now().Sub(start))
			mu.Unlock()
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))
	log.Fatal(http.ListenAndServe("localhost:8675", nil))
}
