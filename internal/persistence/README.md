## persistence

main purpose is that supports different store component.

### Note: 

 While you are implementing the `Repository`, there are some key should be paid extra attention:
 
1. `Pair`'s key should be unique in one namespace.
2. `Container`'s key should be unique in one namespace.
3. `IMigrator` is designed to prepare persistence environment at `cassem`'s first bootstrap, so that do prepare work 
   for your implementation if needed.
4. `IPolicyAdapter` expects you return a `casbin.persist.Adapter` based your storage.