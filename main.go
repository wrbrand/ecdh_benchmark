package main

import (
  "os"
  "io"
  "encoding/json"
  "log"
  "crypto/elliptic"
  "math/big"
  "strings"
  "fmt"
  "time"
  "runtime/pprof"
)

const (
  CurveP256 TLSCurveID = 23
  CurveP384 TLSCurveID = 24
  CurveP521 TLSCurveID = 25
)

type HandshakeParams struct {
  Server ServerParams `json:"server"`
  Client ClientParams `json:"client"`
}

type ServerParams struct {
  Curve   CurveID `json:"curve_id"`
  Public  ECPoint `json:"server_public"`
  Private ECPoint `json:"server_private"`
}

type ClientParams struct {
  Curve   CurveID         `json:"curve_id"`
  Public  ECPoint         `json:"client_public"`
  Private auxCryptoParameter `json:"client_private"`
}

type TLSCurveID uint16

type CurveID struct {
  Name string   `json:"name"`
  Id TLSCurveID `json:"id"`
}

type ECPoint struct {
  X *big.Int  `json:"x"`
  Y *big.Int  `json:"y"`
}

// UnmarshalJSON implements the json.Unmarshler interface
func (p *ECPoint) UnmarshalJSON(b []byte) error {
  aux := struct {
    X *cryptoParameter `json:"x"`
    Y *cryptoParameter `json:"y"`
  }{}
  if err := json.Unmarshal(b, &aux); err != nil {
    return err
  }
  p.X = aux.X.Int
  p.Y = aux.Y.Int
  return nil
}

type cryptoParameter struct {
  *big.Int
}

type auxCryptoParameter struct {
  Raw    []byte `json:"value"`
  Length int    `json:"length"`
}

// UnmarshalJSON implements the json.Unmarshal interface
func (p *cryptoParameter) UnmarshalJSON(b []byte) error {
  var aux auxCryptoParameter
  if err := json.Unmarshal(b, &aux); err != nil {
    return err
  }
  p.Int = new(big.Int)
  p.SetBytes(aux.Raw)
  return nil
}

func curveForCurveID(id TLSCurveID) (elliptic.Curve, bool) {
  switch id {
  case CurveP256:
    return elliptic.P256(), true
  case CurveP384:
    return elliptic.P384(), true
  case CurveP521:
    return elliptic.P521(), true
  default:
    return nil, false
  }
}

func getCPUProfileFile() string {
  return os.Getenv("ECDH_CPUPROFILE")
}

// Replace instances in formatString of {TIMESTAMP} with when formatted as
// YYYYMMDDhhmmss, and {NANOS} as the decimal nanosecond offset.
func getFormattedFile(formatString string, when time.Time) string {
  timestamp := when.Format("20060102150405")
  nanos := fmt.Sprintf("%d", when.Nanosecond())
  ret := strings.Replace(formatString, "{TIMESTAMP}", timestamp, -1)
  ret = strings.Replace(ret, "{NANOS}", nanos, -1)
  return ret
}

// If CPU profiling is enabled (ZGRAB2_CPUPROFILE is not empty), start tracking
// CPU profiling in the configured file. Caller is responsible for invoking
// stopCPUProfile() when finished.
func startCPUProfile() {
  if file := getCPUProfileFile(); file != "" {
    now := time.Now()
    fullFile := getFormattedFile(file, now)
    f, err := os.Create(fullFile)
    if err != nil {
      log.Fatal("could not create CPU profile: ", err)
    }
    if err := pprof.StartCPUProfile(f); err != nil {
      log.Fatal("could not start CPU profile: ", err)
    }
  }
}

// If CPU profiling is enabled (ZGRAB2_CPUPROFILE is not empty), stop profiling
// CPU usage.
func stopCPUProfile() {
  if getCPUProfileFile() != "" {
    pprof.StopCPUProfile()
  }
}

func main() {
  startCPUProfile()
  defer stopCPUProfile()
  params := readParams(os.Args[1])

  failures := 0

  for _, p := range params {
    if curve, ok := curveForCurveID(p.Server.Curve.Id); ok {
      curve.ScalarMult(p.Server.Public.X, p.Server.Public.Y, p.Client.Private.Raw)
    } else {
      failures += 1
      continue
    }
  }

  log.Println("Failures:")
  log.Println(failures)
}

func readParams(filename string) []HandshakeParams {
  f, err := os.Open(filename)

  if err != nil {
    panic(err)
  }

  dec := json.NewDecoder(f)

  var params []HandshakeParams

  for {
    var p HandshakeParams

    if err := dec.Decode(&p); err == io.EOF {
      break
    } else if err != nil{
      log.Fatal(err)
    }

    if p.Server.Curve.Id == 0 {
      continue
    }

    params = append(params, p)
  }

  return params
}
