package main

import (
    "fmt"
    "os"
    "sort"
    "syscall"

    ntfs "essai/ntfstool/core"
)

func do_file(file int64, arg *tActionArg) error {
    var record ntfs.FileRecord

    if err := arg.disk.ReadFileRecord(file, &record); err != nil {
        return err
    }

    fmt.Println()
    fmt.Println("Record:")
    ntfs.PrintStruct(record)

    fmt.Println("Good record:", record.Type.IsGood())

    if record.Type != ntfs.RECTYP_FILE {
        return nil
    }

    _, ok := arg.GetExt("raw")
    if ok {
        return nil
    }

    attributes, err := record.GetAttributes(false)
    if err != nil {
        return err
    }

    _, ok = arg.GetExt("name")
    if ok {
        for _, attr := range attributes {
            if attr.AttributeType == ntfs.ATTR_FILE_NAME {
                desc, err := record.MakeAttributeFromHeader(attr)
                if err != nil {
                    return err
                }

                val, err := arg.disk.GetAttributeValue(desc, true)
                if err != nil {
                    return err
                }

                fmt.Println()
                fmt.Println("Filename:", val.GetFilename())
            }
        }
    }

    index, ok := arg.IntExt("index")
    if ok {
        attr, ok := attributes[int(index)]
        if !ok {
            fmt.Println("Attribute", index, "not found")

            return nil
        }

        switch attr.AttributeType {
        case ntfs.ATTR_INDEX_ROOT, ntfs.ATTR_INDEX_ALLOCATION:
        default:
            fmt.Println("Attribute", index, "is not an index")

            return nil
        }

        desc, err := record.MakeAttributeFromHeader(attr)
        if err != nil {
            return err
        }

        fmt.Println()
        fmt.Println("Attribute:")
        ntfs.PrintStruct(desc.Desc)
        if desc.Name != "" {
            fmt.Println("   Name:", desc.Name)
        }

        val, err := arg.disk.GetAttributeValue(desc, true)
        if err != nil {
            return err
        }

        fmt.Println()
        fmt.Println("Index Content:")

        next_entry, err := val.GetFirstEntry()
        if err != nil {
            return err
        }

        for i := 0; next_entry != nil; i++ {
            entry := next_entry
            next_entry, err = val.GetNextEntry(entry)
            if err != nil {
                return err
            }

            if entry.FileReferenceNumber == 0 {
                fmt.Println(fmt.Sprintf("  - %d : [%v] {%s}", i, entry.FileReferenceNumber, entry.Name))

                continue
            }

            var file_rec ntfs.FileRecord

            if err := arg.disk.ReadFileRecordFromRef(entry.FileReferenceNumber, &file_rec); err != nil {
                if ntfs.IsEof(err) {
                    fmt.Println(fmt.Sprintf("  - %d : [%v] {%s}", i, entry.FileReferenceNumber, entry.Name))

                    continue
                }

                return err
            }

            name, err := arg.disk.GetFileRecordFilename(&file_rec)
            if err != nil {
                return err
            }

            fmt.Println(fmt.Sprintf("  - %d : %s [%v] {%s}", i, name, entry.FileReferenceNumber, entry.Name))
        }

        return nil
    }

    attr_num, ok := arg.IntExt("attribute")
    if ok {
        attr, ok := attributes[int(attr_num)]
        if !ok {
            fmt.Println("Attribute", attr_num, "not found")

            return nil
        }

        desc, err := record.MakeAttributeFromHeader(attr)
        if err != nil {
            return err
        }

        fmt.Println()
        fmt.Println("Attribute:")
        ntfs.PrintStruct(desc.Desc)
        if desc.Name != "" {
            fmt.Println("   Name:", desc.Name)
        }

        _, noread := arg.GetExt("noread")
        _, ask_runlist := arg.GetExt("runlist")

        val, err := arg.disk.GetAttributeValue(desc, !(noread || ask_runlist))
        if err != nil {
            return err
        }

        fmt.Println()
        fmt.Println(fmt.Sprintf("Value (size=%d):", val.Size))
        ntfs.PrintStruct(val.Value)
        fmt.Println("   First LCN:", val.FirstLCN)

        if ask_runlist {
            fmt.Println()
            fmt.Println("Run list:")
            for _, entry := range desc.GetRunList() {
                fmt.Println("  -", entry)
            }

            return nil
        }

        entry_idx, ok := arg.IntExt("block")
        if ok {
            entry, err := val.GetFirstEntry()
            if err != nil {
                return err
            }

            for i := int64(0); i < entry_idx; i++ {
                entry, err = val.GetNextEntry(entry)
                if err != nil {
                    return err
                }
            }

            fmt.Println()
            fmt.Println("Entry:")
            ntfs.PrintStruct(entry.DirectoryEntryExtendedHeader)

            block, err := val.GetIndexBlockFromEntry(entry)
            if err != nil {
                return err
            }

            fmt.Println()
            fmt.Println("Block:")
            ntfs.PrintStruct(block)

            return nil
        }

        save, ok := arg.GetExt("save")
        if ok {
            _, err := os.Stat(save)
            switch err {
            case os.ErrExist, os.ErrNotExist, nil:
            default:
                _, ok := err.(*os.PathError)
                if !ok {
                    return ntfs.WrapError(err)
                }
            }

            var f *os.File

            exists := err != os.ErrNotExist
            if exists {
                perr, ok := err.(*os.PathError)
                if ok {
                    exists = perr.Err != syscall.ENOENT
                }
            }

            if exists {
                f, err = os.OpenFile(save, os.O_TRUNC|os.O_WRONLY, 0660)
            } else {
                f, err = os.OpenFile(save, os.O_CREATE|os.O_WRONLY, 0660)
            }

            if err != nil {
                return ntfs.WrapError(err)
            }

            defer ntfs.DeferedCall(f.Close)

            fmt.Println()
            fmt.Println("Header size:", val.Size-len(val.Data))

            _, err = f.Write(val.Content)
            if err != nil {
                return ntfs.WrapError(err)
            }
        }

        return nil
    }

    attr_list := make([]int, 0)
    for idx := range attributes {
        attr_list = append(attr_list, idx)
    }

    sort.Ints(attr_list)

    for _, idx := range attr_list {
        attr := attributes[idx]

        fmt.Println()
        fmt.Println(fmt.Sprintf("Attribute %d :", idx))
        ntfs.PrintStruct(attr)

        desc, err := record.MakeAttributeFromHeader(attr)
        if err != nil {
            return err
        }

        if desc.Name != "" {
            fmt.Println("   Name:", desc.Name)
        }
    }

    return nil
}
