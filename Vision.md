# Craftmine

This project is a simple minecraft clone to be built in Go using the g3n game engine. It should be focused primarily on performance and modularity, rather than features, so that mods can come through and implement most gameplay later.

For now, the game should contain:
- a minecraft-style main menu, simplified for the time being
  - a single button ("new game") that creates a new world of size 512x512 blocks with a height ranging from 0 to 256.
  - a single button ("resume game") that resumes an existing world and is disabled otherwise
- a world containing grass, dirt, stone, wood, and leaves arranged into hills and trees with a simple terrain generation system.
