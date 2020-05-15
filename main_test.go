package main

import (
  "os"
  "testing"
)

func BenchmarkEllipticScalarMultSerial(b *testing.B) {
  params := readParams(os.Getenv("ECDH_Testfile"))

  for i := 0; i < b.N; i++ {
    for _, p := range params {
      if curve, ok := curveForCurveID(p.Server.Curve.Id); ok {
        curve.ScalarMult(p.Server.Public.X, p.Server.Public.Y, p.Client.Private.Raw)
      } else {
        // Silently skip unsupported curves
        continue
      }
    }
  }
}
