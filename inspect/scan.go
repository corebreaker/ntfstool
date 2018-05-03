package inspect

import (
    "essai/ntfstool/core"
    "fmt"
)

func Scan(name, destination string) error {
    fmt.Println("Scanning...")
    positions, err := FindPositionsWithType(name, core.RECTYP_FILE, core.RECTYP_INDX)
    if err != nil {
        return err
    }

    defer fmt.Println("End.")

    fmt.Println("Writing...")
    out, err := OpenStateWriter(destination)
    if err != nil {
        return err
    }

    defer core.DeferedCall(out.Close)

    for _, position := range positions[0] {
        err := out.Write(&StateFileRecord{
            StateFile: StateFile{
                StateBase: StateBase{
                    Position: position,
                },
            },
        })

        if err != nil {
            return err
        }
    }

    for _, position := range positions[1] {
        err := out.Write(&StateIndexRecord{
            StateFile: StateFile{
                StateBase: StateBase{
                    Position: position,
                },
            },
        })

        if err != nil {
            return err
        }
    }

    return nil
}
