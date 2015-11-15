'use strict';

angular.module('herder', ['herder.controllers']);

angular.module('herder.controllers', []).controller('HerderCtrl', function($scope, $interval) {
    var conn;
    $scope.cattle = [];

    // What do we want to do?
    // We have an array that gets constantly updated. Let's keep track
    // of the changes in the array so that objects that are not present
    // anymore (that got removed on the server side) they don't get
    // removed from our local array.
    // Hint: perhaps using a linked list, using the ID as the key
    // Hint: Perhaps replace the elements that are in a that are not in b with
    //       elements that are in b that are not in a.

    function arrayToObj(arr){
        var result = [];
        arr.forEach(function(el){
            result[el.id] = el;
        });
        return result;
    }
    $scope.$watch('host', function() {
        conn = new WebSocket("ws://" + $scope.host + "/ws");
        conn.onclose = function(evt) {
            data.textContent = 'Connection closed';
        }
        conn.onmessage = function(evt) {
            console.log("=================================================")
            var newData = JSON.parse(evt.data);
            console.log("newData", newData);
            var nrOfInstances = newData.nrOfInstances;

            // the arrray holding this run's cattle
            var newCattle = newData.instances;
            console.log(newCattle);

            // the array holding last run's cattle
            var oldCattle = $scope.cattle;
            console.log("oldCattle", oldCattle);

            if(oldCattle.length == 0){
                $scope.cattle = newData.instances;
            }
            else {
                // holding area for the new data
                var cattle = new Array();

                var newCattleObjs = arrayToObj(newCattle);
                console.log("newCattleObjs", newCattleObjs);

                // go through the old list and update anything that is in the new list
                var cattle = oldCattle.map(function(el){
                    // element exists in new array. Update the values.
                    if(typeof newCattleObjs[el.id] === 'undefined'){
                        return null;
                    } else {
                        return newCattleObjs[el.id];
                    }
                });
                // remove everything from the new array that is in the old array
                newCattle = newCattle.filter(function(el){
                    var found = false;
                    cattle.forEach(function(elOld){
                        if(elOld != null && elOld.id == el.id) {
	                        found = true;
	                        return;
                        }
                    });
                    return !found;
                });
                // newCattle now contains containers that were just started. Let's put them in the open spots in t1
                newCattle.map(function(el){
					var idx = cattle.indexOf(null);
					if(idx != -1){
						cattle[idx] = el;
					} else {
						cattle.push(el);
					}
                });

				cattle = cattle.map(function(el){
					if(el == null) return {"id": "xxxxxxxx", "ops": 30, "max": 0};
					else return el;
				});

				// reduce the length of cattle to the nr of instances
				cattle = cattle.slice(0, nrOfInstances);


                console.log("newCattle", newCattle)
                console.log("cattle", cattle)

                $scope.cattle = cattle;
            }
            $scope.$apply();
        }
    });

    $scope.graydot = function (max) {
        var wi = max / 30 * 150;
        var ma = 75 - (max / 30 * 75);
        return {
           width: wi + "px",
           height: wi + "px",
           margin: ma + "px"
       }
    }

	/*
    yellow  = FE 254, DE 222, 32  32
    red     = FC 252, 0E  14, 39  39
    */
    function coloring(ops, max){
        if (ops < 2) {
            return '#FEDE32';
        }

        var lowColor = [254, 222, 32], hiColor = [252, 14, 39], result = [];
        lowColor.forEach(function(el, lc) {
            result.push(lowColor[lc] + Math.round((hiColor[lc] - lowColor[lc]) * ops / max));
        });

        return 'rgb(' + result.join() + ')';
    }

    $scope.reddot = function (ops, max) {
        var wi = ops / 30 * 150;
        var ma = 75 - (ops / 30 * 75);
        return {
            'background-color': coloring(ops, max),
            width: wi + "px",
            height: wi + "px",
            margin: ma + "px"
        }
    }
});
