import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export function formatBytes(bytes: number): string {
	if (bytes === 0) return "—";
	const units = ["B", "KB", "MB", "GB"];
	const i = Math.floor(Math.log(bytes) / Math.log(1024));
	return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

/** Convert a simple glob pattern (using * and ?) to a case-insensitive RegExp. */
export function globToRegex(glob: string): RegExp {
	let pattern = "";
	for (const ch of glob) {
		switch (ch) {
			case "*":
				pattern += ".*";
				break;
			case "?":
				pattern += ".";
				break;
			// Escape regex special characters
			case ".":
			case "+":
			case "^":
			case "$":
			case "{":
			case "}":
			case "(":
			case ")":
			case "|":
			case "[":
			case "]":
			case "\\":
				pattern += "\\" + ch;
				break;
			default:
				pattern += ch;
		}
	}
	return new RegExp(`^${pattern}$`, "i");
}
