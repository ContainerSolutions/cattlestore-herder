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
            console.log('data received:', JSON.parse(evt.data));
        }
    });
});
