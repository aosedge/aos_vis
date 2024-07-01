# Vehicle Information Service

[![CI](https://github.com/aosedge/aos_vis/workflows/CI/badge.svg)](https://github.com/aosedge/aos_vis/actions?query=workflow%3ACI)
[![codecov](https://codecov.io/gh/aosedge/aos_vis/branch/main/graph/badge.svg?token=h194cLyKqU)](https://codecov.io/gh/aosedge/aos_vis)

aos_vis is the WebSocket server that provides vehicle information data using [W3C protocol](https://www.w3.org/TR/vehicle-information-service/)

## Extendable adapters

aos_vis supports several adapaters. Configuration for each adapter could be specified via a configuration file.

### vinadapter plugin

Provides VIN from the file via VIS path.
Configuration:

```json
{
    "Plugin": "vinadapter",
    "Params": {
        "VISPath": "Attribute.Vehicle.VehicleIdentification.VIN",
        "FilePath": "/var/aos/vis/vin"
    }
}
```

### unitmodeladapter

Provides board model from the file via VIS path.
Configuration:

```json
{
    "Plugin": "unitmodeladapter",
    "Params": {
        "VISPath": "Attribute.Aos.UnitModel",
        "FilePath": "/etc/aos/unit_model"
    }
}
```

### subjectsadapter

Provides Aos subjects from the file via VIS path.
Configuration:

```json
{
    "Plugin": "subjectsadapter",
    "Params": {
        "VISPath": "Attribute.Aos.Subjects",
        "FilePath": "/var/aos/subjects"
    }
}
```

### renesassimulatoradapter

Converts Renesas simulator data to VIS protocol
Configuration:

```json
{
    "Plugin": "renesassimulatoradapter",
    "Params": {
        "ServerURL": ":9000",  // server url to receive vehicle data in Reneseas format
        "Signals": {           // conversion map
            "geometry.coordinates.Latitude": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Latitude",
            "geometry.coordinates.Longitude": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Longitude",
            "geometry.coordinates.Altitude": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Altitude",
            "geometry.coordinates.HorizAccu": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Accuracy"
        }
    }
}
```

### telemetryemulatoradapter

Gets telemetry data from simulator and converts to VIS format.
Configuration:

```json
{
   "Plugin": "telemetryemulatoradapter",
   "Params": {
        "SensorURL": "http://localhost:8800"
    }
}
```

## Build

```bash
go build
```

## Configure

Specify parameters in visconfig.json file:

```json
{
    "ServerURL": ":443",
    "CACert": "/etc/ssl/certs/Aos_Root_CA.pem",
    "VISCert": "data/wwwivi.crt.pem",
    "VISKey": "data/wwwivi.key.pem",
    "PermissionServerURL": "aosiam:8090",
    "Adapters": [
        {
            "Plugin": "vinadapter",
            "Params": {
                "VISPath": "Attribute.Vehicle.VehicleIdentification.VIN",
                "FilePath": "/var/aos/vis/vin"
            }
        },
        {
            "Plugin": "unitmodeladapter",
            "Params": {
                "VISPath": "Attribute.Aos.UnitModel",
                "FilePath": "/etc/aos/board_model"
            }
        },
        {
            "Plugin": "subjectsadapter",
            "Params": {
                "VISPath": "Attribute.Aos.Subjects",
                "FilePath": "/var/aos/subjects"
            }
        },
        {
            "Plugin": "renesassimulatoradapter",
            "Disabled": false,
            "Params": {
                "ServerURL": ":9000",
                "Signals": {
                    "Timestamp": "",
                    "Ver": "",
                    "geometry.coordinates.Latitude": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Latitude",
                    "geometry.coordinates.Longitude": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Longitude",
                    "geometry.coordinates.Altitude": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Altitude",
                    "geometry.coordinates.HorizAccu": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Accuracy",
                    "geometry.coordinates.AltAccu": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.AltitudeAccuracy",
                    "geometry.coordinates.Heading": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Heading",
                    "geometry.coordinates.HeadingAccu": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.HeadingAccuracy",
                    "geometry.coordinates.Speed": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.Speed",
                    "geometry.coordinates.SpeedAccu": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.SpeedAccuracy",
                    "geometry.coordinates.PosType": "Signal.Cabin.Infotainment.Navigation.CurrentLocation.PosType",
                    "RunningStatus.Acceleration.X": "Signal.Vehicle.Acceleration.Longitudinal",
                    "RunningStatus.Acceleration.Y": "Signal.Vehicle.Acceleration.Lateral",
                    "RunningStatus.Acceleration.Z": "Signal.Vehicle.Acceleration.Vertical",
                    "RunningStatus.Vehicle.Speed": "Signal.Vehicle.Speed",
                    "RunningStatus.Engine.Speed": "Signal.Drivetrain.InternalCombustionEngine.Engine.Speed",
                    "RunningStatus.Brake.PedalPosition": "Signal.Chassis.Brake.PedalPosition",
                    "RunningStatus.Accelerator.PedalPosition": "Signal.Chassis.Accelerator.PedalPosition",
                    "RunningStatus.Fuel.Level": "Signal.Drivetrain.FuelSystem.Level",
                    "RunningStatus.SteeringWheel.Angle": "Signal.Chassis.SteeringWheel.Angle",
                    "RunningStatus.Transmission.Gear": "Signal.Drivetrain.Transmission.Gear",
                    "RunningStatus.ParkingBrake.IsEngaged": "Signal.Chassis.ParkingBrake.IsEngaged",
                    "RunningStatus.Battery.Capacity": "Signal.Drivetrain.BatteryManagement.BatteryCapacity",
                    "Body.Door.FrontLeft.IsOpen": "Signal.Cabin.Door.Row1.Left.IsOpen",
                    "Body.Door.FrontLeft.IsLocked": "Signal.Cabin.Door.Row1.Left.IsLocked",
                    "Body.Door.FrontLeft.WindowPosition": "Signal.Cabin.Door.Row1.Left.Window.Position",
                    "Body.Door.FrontLeft.IsMirrorOpen": "Signal.Body.Mirrors.Left.Pan",
                    "Body.Door.FrontRight.IsOpen": "Signal.Cabin.Door.Row1.Right.IsOpen",
                    "Body.Door.FrontRight.IsLocked": "Signal.Cabin.Door.Row1.Right.IsLocked",
                    "Body.Door.FrontRight.WindowPosition": "Signal.Cabin.Door.Row1.Right.Window.Position",
                    "Body.Door.FrontRight.IsMirrorOpen": "Signal.Body.Mirrors.Right.Pan",
                    "Body.Door.RearLeft.IsOpen": "Signal.Cabin.Door.Row2.Left.IsOpen",
                    "Body.Door.RearLeft.IsLocked": "Signal.Cabin.Door.Row2.Left.IsLocked",
                    "Body.Door.RearLeft.WindowPosition": "Signal.Cabin.Door.Row2.Left.Window.Position",
                    "Body.Door.RearRight.IsOpen": "Signal.Cabin.Door.Row2.Right.IsOpen",
                    "Body.Door.RearRight.IsLocked": "Signal.Cabin.Door.Row2.Right.IsLocked",
                    "Body.Door.RearRight.WindowPosition": "Signal.Cabin.Door.Row2.Right.Window.Position",
                    "Body.Bonnet.IsOpen": "Signal.Body.Hood.IsOpen",
                    "Body.Trunk.IsOpen": "Signal.Body.Trunk.IsOpen",
                    "Body.Light.IsHazardOn": "Signal.Body.Lights.IsHazardOn",
                    "Body.Light.IsLowBeamOn": "Signal.Body.Lights.IsLowBeamOn",
                    "Body.Light.IsHighBeamOn": "Signal.Body.Lights.IsHighBeamOn",
                    "Body.Light.IsFrontFogOn": "Signal.Body.Lights.IsFrontFogOn",
                    "Body.Light.IsRearFogOn": "Signal.Body.Lights.IsRearFogOn",
                    "Body.Wiper.Front.Status": "Signal.Body.Windshield.Front.Wiping.Status",
                    "Body.Wiper.Rear.Status": "Signal.Body.Windshield.Rear.Wiping.Status",
                    "Body.FuelCap.IsOpen": "Signal.Body.FuelCap.IsOpen",
                    "Cabin.Seat.FrontLeft.Recline": "Signal.Cabin.Seat.Row1.Pos1.Recline",
                    "Cabin.Seat.FrontLeft.IsSeatbeltOn": "Signal.Cabin.Seat.Row1.Pos1.IsBelted",
                    "Cabin.Seat.FrontLeft.IsAirbagDeployed": "Signal.Cabin.Seat.Row1.Pos1.Airbag.IsDeployed",
                    "Cabin.Seat.FrontRight.Recline": "Signal.Cabin.Seat.Row1.Pos2.Recline",
                    "Cabin.Seat.FrontRight.IsSeatbeltOn": "Signal.Cabin.Seat.Row1.Pos2.IsBelted",
                    "Cabin.Seat.FrontRight.IsAirbagDeployed": "Signal.Cabin.Seat.Row1.Pos2.Airbag.IsDeployed",
                    "Cabin.Seat.RearLeft.Recline": "Signal.Cabin.Seat.Row2.Pos1.Recline",
                    "Cabin.Seat.RearLeft.IsSeatbeltOn": "Signal.Cabin.Seat.Row2.Pos1.IsBelted",
                    "Cabin.Seat.RearRight.Recline": "Signal.Cabin.Seat.Row2.Pos2.Recline",
                    "Cabin.Seat.RearRight.IsSeatbeltOn": "Signal.Cabin.Seat.Row2.Pos2.IsBelted",
                    "Cabin.HVAC.FrontLeft.Temperature": "Signal.Cabin.HVAC.Row1.Left.Temperature",
                    "Cabin.HVAC.FrontRight.Temperature": "Signal.Cabin.HVAC.Row1.Right.Temperature",
                    "Cabin.HVAC.RearLeft.Temperature": "Signal.Cabin.HVAC.Row2.Left.Temperature",
                    "Cabin.HVAC.RearRight.Temperature": "Signal.Cabin.HVAC.Row2.Right.Temperature",
                    "Cabin.HVAC.AmbientAir.Temperature": "Signal.Cabin.HVAC.AmbientAirTemperature",
                    "Cabin.Sunroof.Position": "Signal.Cabin.Sunroof.Position",
                    "DriveTrain.Tire.FrontLeft.Pressure": "Signal.Chassis.Axle.Row1.Wheel.Left.Tire.Pressure",
                    "DriveTrain.Tire.FrontRight.Pressure": "Signal.Chassis.Axle.Row1.Wheel.Right.Tire.Pressure",
                    "DriveTrain.Tire.RearLeft.Pressure": "Signal.Chassis.Axle.Row2.Wheel.Left.Tire.Pressure",
                    "DriveTrain.Tire.RearRight.Pressure": "Signal.Chassis.Axle.Row2.Wheel.Right.Tire.Pressure",
                    "DriveTrain.ADAS.SuspensionMode": "Signal.ADAS.SuspensionMode",
                    "DriveTrain.ADAS.ABS": "Signal.ADAS.ABS.IsEngaged",
                    "DriveTrain.OBD.OilLevel": "Signal.OBD.OilLevel",
                    "DriveTrain.OBD.CoolantTemperature": "Signal.OBD.CoolantTemperature",
                    "Navigation.SpeedLimit": "Signal.Traffic.SpeedLimit",
                    "Navigation.Turn.Direction": "Signal.Traffic.Turn.Direction",
                    "Navigation.Turn.Angle": "Signal.Traffic.Turn.Angle",
                    "Navigation.Curve.Direction": "Signal.Traffic.Curve.Direction",
                    "Navigation.Curve.Level": "Signal.Traffic.Curve.Level",
                    "Event.Unstable": "Private.V2C.Events.Unstable",
                    "Event.RedLight": "Private.V2C.Events.RedLight",
                    "Event.Tire": "Private.V2C.Events.Tire",
                    "Event.Pedestrian": "Private.V2C.Events.Pedestrian",
                    "Event.Accident": "Private.V2C.Events.Accident",
                    "Event.DriverState": "Private.V2C.Events.DriverState",
                    "Event.AOI": "Private.V2C.Events.AOI",
                    "Event.Disconnect": "Private.V2C.Events.Disconnect",
                    "Event.HeavyRain": "Private.V2C.Events.HeavyRain",
                    "Event.Approaching.Front": "Private.V2C.Events.Approaching.Front",
                    "Event.Approaching.Rear": "Private.V2C.Events.Approaching.Rear",
                    "Event.Approaching.RearLeft": "Private.V2C.Events.Approaching.RearLeft",
                    "Event.Approaching.RearRight": "Private.V2C.Events.Approaching.RearRight",
                    "Event.Authentication": "Private.V2C.Events.Authentication",
                    "Emotion.Calm": "Private.V2C.Emotion.Calm",
                    "Emotion.Angry": "Private.V2C.Emotion.Angry",
                    "Emotion.Joy": "Private.V2C.Emotion.Joy",
                    "Emotion.Sorrow": "Private.V2C.Emotion.Sorrow",
                    "Emotion.Excite": "Private.V2C.Emotion.Excite",
                    "Emotion.Level": "Private.V2C.Emotion.Level",
                    "Emotion.PrimaryEmotion": "Private.V2C.Emotion.PrimaryEmotion",
                    "Emotion.Face.Picture": "Private.V2C.Emotion.Face.Picture"
                }
            }
        },
        {
            "Plugin": "telemetryemulatoradapter",
            "Params": {
                "SensorURL": "http://localhost:8800"
            }
        }
    ]
}
