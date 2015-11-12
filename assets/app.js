'use strict';

angular.module('herder', ['herder.controllers']);

angular.module('herder.controllers', []).controller('HerderCtrl', function($scope, $interval) {
    var conn;
    $scope.cattle = [];
    $scope.$watch('host', function() {
        conn = new WebSocket("ws://" + $scope.host + "/ws");
        conn.onclose = function(evt) {
            data.textContent = 'Connection closed';
        }
        conn.onmessage = function(evt) {
            $scope.cattle = JSON.parse(evt.data);
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
