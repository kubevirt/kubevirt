/*
window.EXTERNAL_PROVIDER = {
	    name: 'YOUR PROVIDER NAME',
	    init: function ( providerContext) {return true;}, // return boolean or Promise

	    GET_VM: function ({ lookupId: name }) { return dispatch => {...}; }, // return Promise
	    GET_ALL_VMS: function () {},
	    SHUTDOWN_VM: function ({ name, id }) {},
	    FORCEOFF_VM: function ({ name, id }) {},
	    REBOOT_VM: function ({ name, id }) {},
	    FORCEREBOOT_VM: function ({ name, id }) {},
	    START_VM: function ({ name, id }) {},

	    vmStateMap, // optional map extending VM states for provider's specifics. Will be merged using Object.assign(), see <StateIcon> component

	    canReset: function (state) {return true;}, // return boolean
	    canShutdown: function (state) {return true;},
	    isRunning: function (state) {return true;},
	    canRun: function (state) {return true;},

	    reducer, // optional Redux reducer. If provided, the Redux reducer tree is lazily extended for this new branch (see reducers.es6)  

	    vmTabRenderers: [ // optional, provider-specific array of subtabs rendered for a VM
		            {name: 'Provider-specific subtab', componentFactory: YOUR_COMPONENT_FACTORY}, // see externalComponent.jsx for more info on componentFactory
		          ],

};
*/

var KUBE_SERVER = "192.168.200.2:8184"

function kubectl(args) {
    return cockpit.spawn(["/usr/bin/kubectl", "--server", KUBE_SERVER].concat(args))
}

kubectl.get = (res) => {
    return kubectl(["get", "-ojson", res])
        .then((data) =>  JSON.parse(data))
}

kubectl.create = (obj) => {
    return kubectl(["create", "-f", "-"])
        .input(JSON.stringify(obj))
}

kubectl.delete = (res, name) => {
    return kubectl(["delete", res, name])
}

var TestSubtabReactComponent = null;


var KubeVirtProvider = {
	    name: 'KubeVirt',
	    init: function ( providerContext) {
            // see provider.es6:getProviderContext()
    	    KubeVirtProvider.ctx = providerContext
    	    _lazyCreateReactComponents( providerContext.React )
	        return true;
        }, // return boolean or Promise

	    GET_VM: function (action) {
            console.log(`get vm ${JSON.stringify(action)}`)
		    return (dispatch) => {

		    }
	    }, // return Promise

        GET_ALL_VMS: (action) => {
            console.log(`all vms ${JSON.stringify(action)}`)
            return (dispatch) => {
                kubectl.get("vms")
                .fail((e, d) => console.log("failed to fetch all vms", e, d))
                .then(obj => {
                    console.log("Got", obj)
                    let vmNames = []
                    obj.items.forEach(function(item) {
                        vmNames.push(item.spec.domain.name)
                        dispatch(
                            KubeVirtProvider.ctx.exportedActionCreators.updateOrAddVm(
                            {
                            connectionName: KUBE_SERVER,
                            id: item.metadata.name,
                            osType: 'Linux',
                            autostart: "enabled",
                            state:         item.status.phase.toLowerCase(),
                            name:          item.spec.domain.name,
                            currentMemory: item.spec.domain.memory.value * 1024,
                            vcpus:         4,

                            rssMemory:     item.spec.domain.memory.value * 1024 * Math.random(),
                            cpuUsage:      40
                            }));
                    })
                    // delay a refresh
                    dispatch(KubeVirtProvider.ctx.exportedActionCreators.delayRefresh())
                    // remove undefined domains
                    dispatch(KubeVirtProvider.ctx.exportedActionCreators.deleteUnlistedVMs(KUBE_SERVER, vmNames));
                })
            }
        },

	    canShutdown: function (state) {return true;},
	    SHUTDOWN_VM: function (data) {
            console.log(`shutdown vm ${JSON.stringify(data)}`)
            return (dispatch) => kubectl.delete("vms", data.id)
        },

	    FORCEOFF_VM: function (data) {
            console.log(`forceoff vm ${JSON.stringify(data)}`)
        },

	    canReset: function (state) {return false;}, // return boolean
	    REBOOT_VM: function (data) {},
	    FORCEREBOOT_VM: function (data) {},
	    START_VM: function (action) {
            console.log(`run vm ${JSON.stringify(action)}`)
            return (dispatch) => {}
        },

	    vmStateMap: {
	        pending: {className: 'pficon pficon-warning-triangle-o icon-1x-vms', title: "The VM is pending (scheduled to run)."}
        },

	    isRunning: function (state) {return true;},
	    canRun: function (state) {return false;},

	    reducer: null, // optional Redux reducer. If provided, the Redux reducer tree is lazily extended for this new branch (see reducers.es6)  

        vmTabRenderers: [
          {
            name: 'Migration',
            componentFactory: function () {return TestSubtabReactComponent;}
          },
        ],

};

window.EXTERNAL_PROVIDER = KubeVirtProvider;

function _lazyCreateReactComponents(React) {
  TestSubtabReactComponent = React.createClass(
    {
      propTypes: {
        vm: React.PropTypes.object.isRequired,
      },
      render: function () {
        var vm = this.props.vm;
        return React.createElement('div', {id: 'test-migration-body-'+ vm.name}, 'Content of subtab');
      }
    }
  );
}
