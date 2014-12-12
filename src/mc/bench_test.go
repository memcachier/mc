package mc

import (
  "testing"
)

func BenchmarkSet(b *testing.B) {
  b.StopTimer()
  cn, err := Dial("tcp", mcAddr)
  if err != nil {
    panic(err)
  }

  b.StartTimer()
  defer b.StopTimer()

  for i := 0; i < b.N; i++ {
    _, err = cn.Set("foo", "bar", 0, 0, 0)
    if err != nil {
      panic(err)
    }
  }
}
