import { NextRequest, NextResponse } from "next/server";

const API_BASE = process.env.API_BASE_URL ?? "";

export async function GET(
  _request: NextRequest,
  { params }: { params: { jobId: string } }
) {
  const { jobId } = params;

  const res = await fetch(`${API_BASE}/jobs/${jobId}`, {
    // No caching — this is a live polling endpoint
    cache: "no-store",
  });

  if (!res.ok) {
    return NextResponse.json(
      { error: "Failed to fetch job status" },
      { status: res.status }
    );
  }

  // Return the full record so the client has outputKey, fileType, createdAt etc.
  const data = await res.json();
  return NextResponse.json(data);
}

