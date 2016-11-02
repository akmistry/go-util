package io

import "io"

func WriteMany(w io.Writer, bufs ...[]byte) (int, error) {
        written := 0
        for _, b := range bufs {
                n, err := w.Write(b)
                written += n
                if err != nil {
                        return written, err
                }
        }
        return written, nil
}
