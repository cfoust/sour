#include "game.h"

namespace entities
{
    using namespace game;

    int extraentinfosize() { return 0; }       // size in bytes of what the 2 methods below read/write... so it can be skipped by other games

    void writeent(entity &e, char *buf)   // write any additional data to disk (except for ET_ ents)
    {
    }

    void readent(entity &e, char *buf, int ver)     // read from disk, and init
    {
        if(ver <= 30) switch(e.type)
        {
            case FLAG:
            case MONSTER:
            case TELEDEST:
            case RESPAWNPOINT:
            case BOX:
            case BARREL:
            case PLATFORM:
            case ELEVATOR:
                e.attr1 = (int(e.attr1)+180)%360;
                break;
        }
        if(ver <= 31) switch(e.type)
        {
            case BOX:
            case BARREL:
            case PLATFORM:
            case ELEVATOR:
                int yaw = (int(e.attr1)%360 + 360)%360 + 7;
                e.attr1 = yaw - yaw%15;
                break;
        }
    }
}

