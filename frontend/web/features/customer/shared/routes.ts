const vehiclesPath = "/customer/vehicles";
export const newVehiclePath = `${vehiclesPath}/new`;
export const vehiclePath = (id: string) => `${vehiclesPath}/${id}`;
export const vehicleBookingPath = (id: string) => `${vehiclePath(id)}/booking`;
export const orderPath = (id: string) => `/customer/orders/${id}`;
