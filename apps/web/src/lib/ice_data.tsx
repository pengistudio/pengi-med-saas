export type ICE_TYPES = {
	code: number;
	description: string;
	rate: number;
};

export const ICE_CODES: ICE_TYPES[] = [
	{
		code: 3000,
		description: "ICE No Especificado",
		rate: 0,
	},
	{
		code: 3011,
		description: "ICE Cigarrillos Rubios",
		rate: 0.17,
	},
	{ code: 3021, description: "ICE Cigarrillos Negros", rate: 0.17 },
	{
		code: 3023,
		description:
			"ICE Productos del Tabaco y Sucedáneos del Tabaco excepto Cigarrillos",
		rate: 1.5,
	},
	{ code: 3031, description: "ICE Bebidas Alcohólicas", rate: 0.75 },
	{ code: 3041, description: "ICE Cerveza Industrial", rate: 0.75 },
	{
		code: 3073,
		description: "CE Vehículos Motorizados cuyo PVP de hasta de 20000 USD ",
		rate: 0.05,
	},
	{
		code: 3075,
		description: "CE Vehículos Motorizados PVP entre 30000 y 40000",
		rate: 0.15,
	},
	{
		code: 3077,
		description:
			"ICE Vehículos Motorizados cuyo PVP superior USD 40.000 hasta 50.000 ",
		rate: 0.2,
	},
	{
		code: 3078,
		description:
			"ICE Vehículos Motorizados cuyo PVP superior USD 50.000 hasta 60.000",
		rate: 0.25,
	},
	{
		code: 3079,
		description:
			"ICE Vehículos Motorizados cuyo PVP superior USD 60.000 hasta 70.000",
		rate: 0.3,
	},
	{
		code: 3080,
		description: "ICE Vehículos Motorizados cuyo PVP superior USD 70.000",
		rate: 0.35,
	},
	{
		code: 3081,
		description: "ICE Aviones, Tricares, yates, Barcos de Recreo",
		rate: 0.15,
	},
	{
		code: 3092,
		description: "ICE Servicios de Televisión Prepagada ",
		rate: 0,
	},
	{ code: 3610, description: "ICE Perfumes y Aguas de Tocador ", rate: 0.2 },
	{ code: 3620, description: "ICE Videojuegos ", rate: 0 },
	{
		code: 3630,
		description: "CE Armas de Fuego, Armas deportivas y Municiones ",
		rate: 3,
	},
	{
		code: 3640,
		description: "ICE Focos Incandescentes ",
		rate: 1,
	},
	{
		code: 3660,
		description: "ICE Cuotas Membresías Afiliaciones Acciones",
		rate: 0.35,
	},
	{
		code: 3093,
		description: "ICE Servicios Telefonía Sociedades ",
		rate: 0.15,
	},
	{
		code: 3101,
		description: "ICE Bebidas Energizantes ",
		rate: 0.1,
	},
	{
		code: 3053,
		description: "ICE Bebidas Gaseosas con Alto Contenido de Azúcar",
		rate: 0,
	},
	{
		code: 3054,
		description: "ICE Bebidas Gaseosas con Bajo Contenido de Azúcar",
		rate: 0.1,
	},
	{ code: 3111, description: "ICE Bebidas No Alcohólicas", rate: 0 },
	{ code: 3043, description: "ICE Cerveza Artesanal", rate: 0 },
	{ code: 3033, description: "ICE Alcohol", rate: 0.75 },
	{
		code: 3671,
		description: "ICE calefones y sistemas de calentamiento de agua a gas SRI ",
		rate: 1,
	},
	{
		code: 3684,
		description:
			"ICE vehículos motorizados camionetas y de rescate cuyo PVP sea hasta DE 30.000 USD",
		rate: 0.05,
	},
	{
		code: 3686,
		description:
			"ICE vehículos motorizados excepto camionetas y de rescate cuyo PVP sea superior USD 20.000 hasta DE 30.000",
		rate: 0.1,
	},
	{
		code: 3388,
		description: "ICE vehículos híbridos cuyo PVP sea de hasta USD. 35.000 ",
		rate: 0,
	},
	{
		code: 3691,
		description:
			"CE vehículos híbridos cuyo PVP superior USD. 35.000 hasta 40.000",
		rate: 0.08,
	},
	{
		code: 3692,
		description:
			"ICE vehículos híbridos cuyo PVP superior USD. 40.000 hasta 50.000 ",
		rate: 0.14,
	},
	{
		code: 3695,
		description:
			"ICE vehículos híbridos cuyo PVP superior USD. 50.000 hasta 60.000",
		rate: 0.2,
	},
	{
		code: 3696,
		description:
			"ICE vehículos híbridos cuyo PVP superior USD. 60.000 hasta 70.000",
		rate: 0.26,
	},
	{
		code: 3698,
		description: "ICE vehículos híbridos cuyo PVP superior a USD 70.000 ",
		rate: 0.32,
	},
	{
		code: 3682,
		description: "ICE consumibles tabaco calentado y líquidos con nicotina SRI",
		rate: 1.5,
	},
	{
		code: 3681,
		description: "ICE servicios de telefonía móvil personas naturales",
		rate: 0,
	},
	{
		code: 3680,
		description: "ICE fundas plásticas ",
		rate: 0,
	},
	{ code: 3533, description: "ICE Import. Bebidas Alcohólicas ", rate: 0.75 },
	{ code: 3541, description: "ICE Cerveza Gran Escala CAE ", rate: 0.75 },
	{ code: 3542, description: "ICE Cigarrillos Rubios CAE ", rate: 0 },
	{ code: 3543, description: "ICE Cigarrillos Negros CAE ", rate: 0 },
	{
		code: 3544,
		description:
			"ICE Productos del Tabaco y Sucedáneos del Tabaco Excepto Cigarrillos CAE ",
		rate: 1.5,
	},
	{ code: 3581, description: "CE Aeronaves CAE ", rate: 0.15 },
	{
		code: 3582,
		description:
			"ICE Aviones, Avionetas y Helicópteros Exct. Aquellos destinados Al Trans. CAE ",
		rate: 0.15,
	},
	{
		code: 3710,
		description: "ICE Perfumes Aguas de Tocador CAE ",
		rate: 0.2,
	},
	{
		code: 3720,
		description: "ICE Video Juegos CAE ",
		rate: 0.35,
	},
	{
		code: 3730,
		description:
			"ICE Importaciones Armas de Fuego, Armas deportivas y Municiones CAE ",
		rate: 3,
	},
	{
		code: 3740,
		description: "ICE Focos Incandescentes CAE ",
		rate: 1,
	},
	{
		code: 3871,
		description:
			"ICE-vehículos motorizados cuyo PVP SEA hasta de 20000 USD SENAE ",
		rate: 0.05,
	},
	{
		code: 3873,
		description: "ICE-vehículos motorizados PVP entre 30000 Y 40000 SENAE ",
		rate: 0.15,
	},
	{
		code: 3874,
		description:
			"ICE-vehículos motorizados cuyo PVP superior USD 40.000 hasta 50.000 SENAE",
		rate: 0.2,
	},
	{
		code: 3875,
		description:
			"ICE-vehículos motorizados cuyo PVP superior USD 50.000 hasta 60.000 SENAE",
		rate: 0.25,
	},
	{
		code: 3876,
		description:
			"ICE-vehículos motorizados cuyo PVP superior USD 60.000 hasta 70.000 SENAE",
		rate: 0.3,
	},
	{
		code: 3877,
		description: "ICE-vehículos motorizados cuyo PVP superior USD 70.000 SENAE",
		rate: 0.35,
	},
	{
		code: 3878,
		description: "ICE-Aviones, Tricares, Yates, Barcos de Rec SENAE ",
		rate: 0.15,
	},
	{
		code: 3601,
		description: "ICE Bebidas Energizantes SENAE ",
		rate: 0.1,
	},
	{
		code: 3552,
		description: "ICE bebidas gaseosas con alto contenido de azúcar SENAE",
		rate: 0,
	},
	{
		code: 3553,
		description: "ICE bebidas gaseosas con bajo contenido de azúcar SENAE",
		rate: 0.1,
	},
	{
		code: 3602,
		description: "CE bebidas no alcohólicas SENAE",
		rate: 0,
	},
	{
		code: 3545,
		description: "ICE cerveza artesanal SENAE ",
		rate: 0.75,
	},
	{
		code: 3532,
		description: "ICE Import. alcohol SENAE ",
		rate: 0.75,
	},
	{
		code: 3771,
		description:
			"ICE calefones y sistemas de calentamiento de agua a gas SENAE",
		rate: 1,
	},
	{
		code: 3685,
		description:
			"ICE vehículos motorizados camionetas y de rescate PVP sea hasta DE 30.000 USD SENAE",
		rate: 0.05,
	},
	{
		code: 3687,
		description:
			"ICE vehículos motorizados excepto camionetas y de rescate cuyo PVP sea superior USD 20.000 hasta DE 30.000 SENAE",
		rate: 0.1,
	},
	{
		code: 3689,
		description:
			"ICE vehículos híbridos cuyo PVP sea de hasta USD. 35.000 SENAE",
		rate: 0,
	},
	{
		code: 3690,
		description:
			"ICE vehículos híbridos cuyo PVP superior USD. 35.000 hasta 40.000 SENAE",
		rate: 0.08,
	},
	{
		code: 3693,
		description:
			"ICE vehículos híbridos cuyo PVP superior USD. 40.000 hasta 50.000 SENAE",
		rate: 0.14,
	},
	{
		code: 3694,
		description:
			"ICE vehículos híbridos cuyo PVP superior USD. 50.000 hasta 60.000 SENAE",
		rate: 0.2,
	},
	{
		code: 3697,
		description:
			"ICE vehículos híbridos cuyo PVP superior USD. 60.000 hasta 70.000 SENAE",
		rate: 0.26,
	},
	{
		code: 3699,
		description: "ICE vehículos híbridos cuyo PVP superior a USD 70.000 SENAE",
		rate: 0.32,
	},
	{
		code: 3683,
		description:
			"ICE consumibles tabaco calentado y líquidos con nicotina SENAE",
		rate: 1.5,
	},
];
