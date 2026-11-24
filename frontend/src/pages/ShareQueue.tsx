import { Check, Copy, ExternalLink } from "lucide-react";
import { useState } from "react";
import { generateQRImage } from "@/api/queues";
import type { BusinessMembership } from "../api/types";
import { Button, Card, InlineError, Input } from "../components/foundation";
import { useToast } from "../components/Toast";
import { useSound } from "../state/SoundContext";
import { BusinessOperationsPage } from "./BusinessOperations";

export default function ShareQueue() {
	return (
		<BusinessOperationsPage title="Share queue">
			{(business) => <Share business={business} />}
		</BusinessOperationsPage>
	);
}

function Share({ business }: { business: BusinessMembership }) {
	const toast = useToast();
	const sound = useSound();
	const [copied, setCopied] = useState(false);
	const [error, setError] = useState("");
	const url = `${window.location.origin}/queuelite/qr/${encodeURIComponent(business.id)}`;
	async function copy() {
		setError("");
		try {
			await navigator.clipboard.writeText(url);
			setCopied(true);
			sound.stepSuccess();
			toast.success("Queue link copied to clipboard.");
			window.setTimeout(() => setCopied(false), 2000);
		} catch {
			sound.softError();
			setError(
				"Copying was blocked by your browser. Select and copy the link manually.",
			);
		}
	}
	return (
		<div className="ql-stack">
			<Card className="ql-share-card">
				<h2>Customer queue link</h2>
				<p className="ql-muted">
					Put this link in a QR code, message, or sign. It opens the queue entry
					flow for this business and never contains customer credentials.
				</p>
				<div className="ql-copy-row">
					<Input
						label="Queue entry URL"
						value={url}
						readOnly
						onFocus={(event) => event.currentTarget.select()}
					/>
					<Button
						onClick={() => {
							void copy();
						}}
					>
						<span key={copied ? "copied" : "copy"} className="ql-icon-pop">
							{copied ? (
								<Check aria-hidden="true" />
							) : (
								<Copy aria-hidden="true" />
							)}
						</span>
						{copied ? "Copied" : "Copy link"}
					</Button>
				</div>
				{error && <InlineError>{error}</InlineError>}
				<div className="ql-actions-row">
					<Button asChild variant="secondary">
						<a href={url} target="_blank" rel="noreferrer">
							Preview queue entry <ExternalLink size={16} aria-hidden="true" />
						</a>
					</Button>
					<form
						onSubmit={(event) => {
							event.preventDefault();
							generateQRImage(business.id);
						}}
					>
						<Button type="submit">Generate QR</Button>
					</form>
				</div>
			</Card>
			<p className="ql-meta ql-share-note">
				QueueLite does not send this link to an external QR service. You can use
				the copied URL with your preferred local print or QR workflow.
			</p>
		</div>
	);
}
