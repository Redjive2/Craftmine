# Craftmine

This project is a simple minecraft clone to be built in Go using the g3n game engine. 
It should be focused primarily on performance and modularity, rather than features, so that mods can come through and implement most gameplay later.

For now, the game should contain:
- a minecraft-style main menu, simplified for the time being
  - a single button ("new game") that creates a new world of size 512x512 blocks with a height ranging from 0 to 256.
  - a single button ("resume game") that resumes an existing world and is disabled otherwise
- a world containing grass, dirt, stone, wood, and leaves arranged into hills and trees with a simple terrain generation system.

The code should be written in a plain and extremely careful style where all functions are either pure computational functions or mostly dumb rendering functions. 
Keep names plain and clear. Do not rely on abbreviations outside of well-known industry and Go standards. Names should be short where possible but plain and full english words where an abbreviated like `i` or `val` is not very common.
Reasonable logging and effectful error handling does not qualify as a side effect for our purposes.
All code should be tested either in batch tests that test all expected paths of a function at once or integration tests that do the same by engineering that situation in the program. Testing should be aggressive and regularly checked for correctness.
Do not pass functions around. With the *sole* exception of the `Impl` struct, functions are to be stored nowhere (unless strictly necessary for some reason or required by an external tool like a slices.Map call or something)
Functions, files, and modules should never be huge or carry multiple concerns, but should not be completely atomized either. Ensure functions are their proper length.
Aggressively parse/validate all input. Implement standardized and detailed logging and error handling to this end.
Modules should be defined as either utility modules (like logging, for example) that anyone can use or primary modules that are independent of all but their submodules. Keep primary modules separate, funnel shared concerns through the top level. Do NOT resort to weird wiring; write the code such that this architecture is normal.
Keep mutations inside a function to the starts and ends of blocks where possible, but this is more suggestion than rule.
*No* global mutable state is to be kept *anywhere*  Instead, any mutable state is to be kept in a struct called `Model` inside the file `model.go` per module, passed as an **argument**, not a receiver, to each function. 
All fields in each `Model` are to be private, only accessible via accessors (simply `.Field()` for both normal and computed fields, NOT `.GetField()` or any equivalents.)
All functions should be attached to a struct, `Impl`, which contains only the functions and *no* fields. It must be backed by an interface.
All *immutable* state should also be kept in the model. Mutable state should be editable via a `SetField()` function or, where preferable, a special function which only allows valid state transitions. 
Nowhere in the codebase are pointers to be used except where necessary. If, for performance or engine reasons, one must be passed - it may *not* be written to. Additionally, all data types should be trivially serializable where possible.
Try to only pass the functions and arguments strictly required for a function, to a function. This may require defining sub-model and sub-implementation interfaces to pass around.
Finally, never create 'god objects'. That is, avoid structs that contain anything and everything. Split `Model`s up as necessary. Keep function arguments outside the Model. Use other struct types to supplement when mutability is not a concern. 

Please create and spawn a new ephemeral agent, the Enforcer, that reviews the codebase for style or architectural violations and remedies them (or spawns a polecat if it's more than a few lines.) Spawn this guy periodically.

In the future, the game will contain extensive shader work, ecological systems, mechanical systems (which do not necessarily follow a naive block-coordinate system) and so on, so bear that in mind in designing the implementation of the game.
