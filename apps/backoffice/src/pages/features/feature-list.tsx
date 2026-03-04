import { Pencil, Plus, Trash2 } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	deleteFeature,
	type Feature,
	getFeatures,
} from "@/api/feature-service";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { useText } from "@/hooks/use-text";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const FeatureList = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [features, setFeatures] = React.useState<Feature[]>([]);
	const [loading, setLoading] = React.useState(true);

	const fetch = React.useCallback(async () => {
		setLoading(true);
		const res = await getFeatures();
		if (res.success && res.data) setFeatures(res.data as Feature[]);
		setLoading(false);
	}, []);

	React.useEffect(() => {
		fetch();
	}, [fetch]);

	const handleDelete = async (id: number) => {
		const res = await deleteFeature(id);
		if (res.success) fetch();
	};

	return (
		<DashboardLayout>
			<div className="space-y-6">
				<div className="flex items-center justify-between">
					<h1 className="text-2xl font-bold tracking-tight">
						{textGet("backoffice.features.title")}
					</h1>
					<Button onClick={() => navigate("/features/create")}>
						<Plus className="h-4 w-4 mr-2" />
						{textGet("backoffice.features.create")}
					</Button>
				</div>
				<Card>
					<CardHeader>
						<CardTitle>{textGet("backoffice.features.list.title")}</CardTitle>
						<CardDescription>
							{textGet("backoffice.features.list.description")}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<p className="text-sm text-muted-foreground py-8 text-center animate-pulse">
								{textGet("backoffice.companies.loading")}
							</p>
						) : features.length === 0 ? (
							<p className="text-sm text-muted-foreground py-8 text-center">
								{textGet("backoffice.features.empty")}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											{textGet("backoffice.features.col.code")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.features.col.name")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.features.col.permissions")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("table.actions")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{features.map((f) => (
										<TableRow key={f.ID}>
											<TableCell className="font-mono text-sm">
												{f.code}
											</TableCell>
											<TableCell className="font-medium">{f.name}</TableCell>
											<TableCell>
												<span className="inline-flex items-center rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
													{f.permissions?.length ?? 0}
												</span>
											</TableCell>
											<TableCell className="text-right">
												<div className="flex items-center justify-end gap-2">
													<Button
														variant="ghost"
														size="icon"
														onClick={() => navigate(`/features/edit/${f.ID}`)}
													>
														<Pencil className="h-4 w-4" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														onClick={() => handleDelete(f.ID)}
													>
														<Trash2 className="h-4 w-4 text-destructive" />
													</Button>
												</div>
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>
			</div>
		</DashboardLayout>
	);
};

export default FeatureList;
