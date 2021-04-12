## myraft

The component which take care of how to broadcast command between `cassemd` peer nodes. It holds some important 
information: `Nodes`.


`cassemd` component has some status:
* Starting: `cassemd` starting all component, so it could not receive handle request.
* Up: could work normally.
* Shutting: `Shutting` down, so it would refuse all coming requests.