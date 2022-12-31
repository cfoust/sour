package maps

func EmptyMap(scale int) *Cube {
    root := NewCubes(F_EMPTY, MAT_AIR)

    for i := 0; i < 4; i++ {
        root.Children[i].SolidFaces()
    }

    //if(worldsize > 0x1000) splitocta(worldroot, worldsize>>1);

    return root
}
