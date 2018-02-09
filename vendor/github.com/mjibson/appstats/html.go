/*
 * Copyright (c) 2013 Matt Jibson <matt.jibson@gmail.com>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

/*
 * These are modified from the python appstats implementation.
 * If the above license infringes in some way on the original owner's
 * copyright, I will change it.
 */

package appstats

const htmlBase = `
{{ define "top" }}<!DOCTYPE html>
<html lang="en">
<head>
  <meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
  <style>
    @import "static/appstats_css.css";
  </style>
  <title>Appstats - {{.Env.APPLICATION_ID}}</title>
{{ end }}

{{ define "body" }}
</head>
<body>
  <div class="g-doc">
    {{/* Header begin */}}
    <div id="hd" class="g-section">
      <div class="g-section">
        <a href="."><img id="ae-logo" src="static/app_engine_logo_sm.gif"
          width="176" height="30" alt="Google App Engine" border="0"></a>
      </div>
      <div id="ae-appbar-lrg" class="g-section">
        <div class="g-section g-tpl-50-50 g-split">
          <div class="g-unit g-first">
            <h1>Application Stats for {{.Env.APPLICATION_ID}}</h1>
          </div>
          <div class="g-unit">
            All costs displayed in micropennies (1 dollar equals 100 pennies, 1 penny equals 1 million micropennies)
          </div>
        </div>
      </div>
    </div>
    {{/* Header end */}}
    {{/* Body begin */}}
    <div id="bd" class="g-section">
      {{/* Content begin */}}
      <div>
{{ end }}

{{ define "end" }}
      </div>
      {{/* Content end */}}
    </div>
    {{/* Body end */}}
  </div>
<script src="static/appstats_js.js"></script>
{{ end }}

{{ define "footer" }}
</body>
</html>
{{ end }}
`

const htmlMain = `
{{ define "main" }}
{{ template "top" . }}
{{ template "body" . }}

<form id="ae-stats-refresh" action=".">
  <button id="ae-refresh">Refresh Now</button>
</form>

{{ if .Requests }}
<div class="g-section g-tpl-33-67">
  <div class="g-unit g-first">
    {{/* RPC stats table begin */}}
    <div class="ae-table-wrapper-left">
      <div class="ae-table-title">
        <div class="g-section g-tpl-50-50 g-split">
          <div class="g-unit g-first"><h2>RPC Stats</h2></div>
          <div id="ae-rpc-expand-all" class="g-unit"></div>
        </div>
      </div>
      <table cellspacing="0" cellpadding="0" class="ae-table ae-stripe" id="ae-table-rpc">
        <colgroup>
          <col id="ae-rpc-label-col">
          <col id="ae-rpc-stats-col">
        </colgroup>
        <thead>
          <tr>
            <th>RPC</th>
            <th>Count</th>
            <th>Cost</th>
            <th>Cost&nbsp;%</th>
          </tr>
        </thead>
        {{ range $index, $item := .AllStatsByCount }}
        <tbody>
          <tr>
            <td>
              <span class="goog-inline-block ae-zippy ae-zippy-expand" id="ae-rpc-expand-{{$index}}"></span>
              {{$item.Name}}
            </td>
            <td>{{$item.Count}}</td>
            <td title="">{{$item.Cost}}</td>
            <td>{{/*$item.CostPct*/}}</td>
          </tr>
        </tbody>
        <tbody class="ae-rpc-detail" id="ae-rpc-expand-{{$index}}-detail">
          {{ range $subitem := $item.SubStats }}
          <tr>
            <td class="rpc-req">{{$subitem.Name}}</td>
            <td>{{$subitem.Count}}</td>
            <td title="">{{$subitem.Cost}}</td>
            <td>{{/*$subitem.CostPct*/}}</td>
          </tr>
          {{ end }}
        </tbody>
        {{ end }}
      </table>
    </div>
    {{/* RPC stats table end */}}
  </div>
  <div class="g-unit">
    {{/* Path stats table begin */}}
    <div class="ae-table-wrapper-right">
      <div class="ae-table-title">
        <div class="g-section g-tpl-50-50 g-split">
          <div class="g-unit g-first"><h2>Path Stats</h2></div>
          <div class="g-unit" id="ae-path-expand-all"></div>
        </div>
      </div>
      <table cellspacing="0" cellpadding="0" class="ae-table" id="ae-table-path">
        <colgroup>
          <col id="ae-path-label-col">
          <col id="ae-path-rpcs-col">
          <col id="ae-path-reqs-col">
          <col id="ae-path-stats-col">
        </colgroup>
        <thead>
          <tr>
            <th>Path</th>
            <th>#RPCs</th>
            <th>Cost</th>
            <th>Cost%</th>
            <th>#Requests</th>
            <th>Most Recent requests</th>
          </tr>
        </thead>
        {{ range $index, $item := .PathStatsByCount }}
        <tr>
          <td>
            <span class="goog-inline-block ae-zippy ae-zippy-expand" id="ae-path-expand-{{$index}}"></span>
            {{$item.Name}}
          </td>
          <td>
            {{$item.Count}}
          </td>
          <td title="">{{$item.Cost}}</td>
          <td>{{/*$item.CostPct*/}}</td>
          <td>{{$item.Requests}}</td>
          <td>
            {{ range $index, $element := $item.RecentReqs }}
                {{ if lt $index 10 }}
                    <a href="#req-{{$element}}">({{$element}})</a>
                {{ end }}
                {{ if eq $index 10 }}
                    ...
                {{ end }}
            {{ end }}
          </td>
          <tbody class="path path-{{$index}}">
            {{ range $subitem := $item.SubStats }}
            <tr>
              <td class="rpc-req">{{$subitem.Name}}</td>
              <td>{{$subitem.Count}}</td>
              <td title="">{{$subitem.Cost}}</td>
              <td>{{/*$subitem.CostPct*/}}</td>
              <td></td>
              <td></td>
            </tr>
            {{ end }}
          </tbody>
        {{ end }}
      </table>
    </div>
    {{/* Path stats table end */}}
  </div>
</div>
<div id="ae-req-history">
  <div class="ae-table-title">
    <div class="g-section g-tpl-50-50 g-split">
      <div class="g-unit g-first"><h2>Requests History</h2></div>
      <div class="g-unit" id="ae-request-expand-all"></div>
    </div>
  </div>

  <table cellspacing="0" cellpadding="0" class="ae-table" id='ae-table-request'>
    <colgroup>
      <col id="ae-reqs-label-col">
    </colgroup>
    <thead>
      <tr>
        <th colspan="4">Request</th>
      </tr>
    </thead>
    {{ range $index, $r := .Requests }}
    <tbody>
      <tr>
        <td colspan="4" class="ae-hanging-indent">
          <span class="goog-inline-block ae-zippy ae-zippy-expand" id="ae-path-requests-{{$index}}"></span>
          ({{$index}})
          <a name="req-{{$index}}" href="details?time={{$r.RequestStats.Start.Nanosecond}}" class="ae-stats-request-link">
            {{$r.RequestStats.Start}}
            "{{$r.RequestStats.Method}}
            {{$r.RequestStats.Path}}{{if $r.RequestStats.Query}}?{{$r.RequestStats.Query}}{{end}}"
            {{if $r.RequestStats.Status}}{{$r.RequestStats.Status}}{{end}}
          </a>
          real={{$r.RequestStats.Duration}}
          {{/*
          overhead={{$r.overhead_walltime_milliseconds}}ms
          ({{$r.combined_rpc_count}} RPC{{$r.combined_rpc_count}},
            billed_ops=[{{$r.combined_rpc_billed_ops}}])
          */}}
          ({{$r.RequestStats.RPCStats | len}} RPCs,
            cost={{$r.RequestStats.Cost}})
        </td>
      </tr>
    </tbody>
    <tbody class="reqon" id="ae-path-requests-{{$index}}-tbody">
      {{ range $item := $r.SubStats }}
      <tr>
        <td class="rpc-req">{{$item.Name}}</td>
        <td>{{$item.Count}}</td>

        <td>{{$item.Cost}}</td>
        {{/*<td>{{$item.total_billed_ops_str}}</td>*/}}
      </tr>
      {{ end }}
    </tbody>
    {{ end }}
  </table>
</div>
{{ else }}
<div>
  No requests have been recorded yet.  While it is possible that you
  simply need to wait until your server receives some requests, this
  is often caused by a configuration problem.
  <a href="http://godoc.org/github.com/mjibson/appstats"
  >Learn more</a>
</div>
{{ end }}

{{ template "end" . }}

<script>
  var z1 = new ae.Stats.MakeZippys('ae-table-rpc', 'ae-rpc-expand-all');
  var z2 = new ae.Stats.MakeZippys('ae-table-path', 'ae-path-expand-all');
  var z3 = new ae.Stats.MakeZippys('ae-table-request', 'ae-request-expand-all');
</script>

{{ template "footer" . }}
{{ end }}
`

const htmlDetails = `
{{ define "details" }}
{{ template "top" . }}
{{ template "body" . }}

{{ if not .Record }}
  <p>Invalid or stale record key!</p>
{{ else }}
  <div class="g-section" id="ae-stats-summary">
    <dl>
      <dt>
        <span class="ae-stats-date">{{.Record.Start}}</span><br>
        <span class="ae-stats-response ae-stats-response-{{.Record.Status}}">
          {{.Record.Status}}
        </span>
      </dt>
      <dd>
        <a {{ if eq .Record.Method "GET" }}target="_new" title="Resubmit the original request to the server" href="{{.Record.Path}}?{{.Record.Query}}" {{ end }}>
          {{.Record.Method}}  {{.Record.Path}}{{if .Record.Query}}?{{.Record.Query}}{{end}}
        </a>
        <br>
        {{.Record.User}}{{ if .Record.Admin }}*{{ end }}
        real={{.Record.Duration}}
        cost={{.Record.Cost}}
        {{/*
        overhead={{.Record.overhead_walltime_milliseconds}}ms
        <br>
        billed_ops={{.Record.combined_rpc_billed_ops}}
        */}}
      </dd>
    </dl>
  </div>

  <div id="ae-stats-details-timeline">
    <h2>Timeline</h2>
    <div id="ae-body-timeline">
      <div id="ae-rpc-chart">[Chart goes here]</div>
    </div>
    {{ if .Record.RPCStats }}
      <div id="ae-rpc-traces">
        <div class="ae-table-title">
          <div class="g-section g-tpl-50-50 g-split">
            <div class="g-unit g-first"><h2>RPC Call Traces</h2></div>
            <div class="g-unit" id="ae-rpc-expand-all"></div>
          </div>
        </div>
        <table cellspacing="0" cellpadding="0" class="ae-table" id="ae-table-rpc">
          <thead>
            <tr>
              <th>RPC</th>
            </tr>
          </thead>
          {{ range $index, $t := .Record.RPCStats }}
          <tbody id="rpc{{$index}}">
            <tr>
              <td>
                <span class="goog-inline-block ae-zippy ae-zippy-expand" id="ae-path-requests-{{$index}}"></span>
                @{{$t.Offset}}
                <b>{{$t.Name}}</b>
                real={{$t.Duration}}
                cost={{$t.Cost}}
                {{/*
                billed_ops=[{{t.billed_ops_str}}]
                */}}
              </td>
            </tr>
          </tbody>
          <tbody>
            {{ if $t.In }}
            <tr>
              <td style="padding-left: 20px"><b>Request:</b> {{$t.Request}}</td>
            </tr>
            {{ end }}
            {{ if $t.Out }}
            <tr>
              <td style="padding-left: 20px"><b>Response:</b> {{$t.Response}}</td>
            </tr>
            {{ end }}
            {{ if $t.StackData }}
            <tr>
              <td style="padding-left: 20px"><b>Stack:</b></td>
            </tr>
            {{ range $stackindex, $f := $t.Stack }}
              <tr>
                <td style="padding-left: 40px">
                  <span  style="padding-left: 12px; text-indent: -12px" class="goog-inline-block ae-zippy-expand" id="ae-head-stack-{{$index}}-{{$stackindex}}">&nbsp;</span>
                  <a href="file?f={{ $f.Location }}&n={{ $f.Lineno }}#n{{ add $f.Lineno -10 }}">{{ $f.Location }}:{{ $f.Lineno }}</a> {{ $f.Call }}
                </td>
              </tr>
              {{/*
              {{ if $f.variables_size }}
                <tr id="ae-body-stack-{{forloop.parentloop.counter}}-{{forloop.counter}}">
                  <td style="padding-left: 60px">{{ for item in f.variables_list }}{{item.key}} = {{item.value}}<br>{{ end }}
                  </td>
                </tr>
              {{ end }}{{# f.variables_size #}
              */}}
            {{ end }}{{/* t.call_stack_list */}}
            {{ end }}{{/* t.call_stack_size */}}
          </tbody>
          {{ end }}{{/* .Record.individual_stats_list */}}
        </table>
      </div>
    {{ end }}{{/* traces */}}
  </div>

  {{ if .AllStatsByCount }}
    <div id="ae-stats-details-rpcstats">
      <h2>RPC Stats</h2>
      <table cellspacing="0" cellpadding="0" class="ae-table" id="ae-table-rpcstats">
        <tbody>
          <tr>
            <td>service.call</td>
            <td align="right">#RPCs</td>
            <td align="right">real time</td>
            <td align="right">Cost</td>
            <td align="right">Billed Ops</td>
          </tr>
          {{ range $item := .AllStatsByCount }}
          <tr>
            <td>{{$item.Name}}</td>
            <td align="right">{{$item.Count}}</td>
            <td align="right">{{$item.Duration}}</td>
            <td align="right">{{$item.Cost}}</td>
            <td align="right"></td>
            <td align="right"></td>
          </tr>
          {{ end }}
        </tbody>
      </table>
    </div>
  {{ end }}{{/* rpcstats_by_count */}}

  {{ if .Header }}
    <div id="ae-stats-details-cgienv">
      <h2>CGI Environment</h2>
      <table cellspacing="0" cellpadding="0" class="ae-table" id="ae-table-cgienv">
        <tbody>
          {{ range $key, $value := .Header }}
          <tr>
            <td align="right" valign="top">{{$key}}=</td>
            <td valign="top">{{$value}}</td>
          </tr>
          {{ end }}
        </tbody>
      </table>
    </div>
  {{ end }}{{/* .Header */}}

{{ end }}

{{ template "end" . }}

<script>
var rpcZippyMaker = new ae.Stats.MakeZippys('ae-table-rpc',
    'ae-rpc-expand-all');
var rpcZippys = rpcZippyMaker.getZippys();
{{ range $index, $t := .Record.RPCStats }}
  {{ range $stackindex, $f := $t.Stack }}
    {{/* if f.variables_size }}
      new goog.ui.Zippy(
          'ae-head-stack-{{forloop.parentloop.counter}}-{{forloop.counter}}',
          'ae-body-stack-{{forloop.parentloop.counter}}-{{forloop.counter}}',
          false);
    {{ end */}}
  {{ end }}
{{ end }}
</script>
<script>
var detailsTabs_ = new ae.Stats.Details.Tabs(['timeline', 'rpcstats',
    'cgienv', 'syspath']);
</script>
<script>
function timelineClickHandler(zippyIndex, hash) {
  rpcZippyMaker.getExpandCollapse().setExpanded(false);
  rpcZippys[zippyIndex].setExpanded(true);

  var headlineIndex = parseInt(zippyIndex, 10) + 1;
  var zippyLine = document.getElementById('ae-path-requests-' + headlineIndex);
  zippyLine.scrollIntoView(true);
}
function renderChart() {
  var chart = new Gantt();
  {{ range $index, $t := .Record.RPCStats }}
    chart.add_bar('{{$t.Name}}',
        {{$t.Offset.Seconds}} * 1000, {{$t.Duration.Seconds}} * 1000,
        0,
        '{{$t.Duration}}',
        'javascript:timelineClickHandler(\'{{$index}}\');');
  {{ end }}

  chart.add_bar('<b>RPC Total</b>', 0, {{.Real.Seconds}} * 1000, 0,
      '{{.Real}}',
      '');
  chart.add_bar('<b>Grand Total</b>', 0, {{.Record.Duration.Seconds}} * 1000, 0,
      '{{.Record.Duration}}', '');
  document.getElementById('ae-rpc-chart').innerHTML = chart.draw();
}
renderChart();
</script>

{{ template "footer" . }}
{{ end }}
`
const htmlFile = `
{{ define "file" }}
{{ template "top" . }}
{{ template "body" . }}

<h1>{{.Filename}}</h1>
<a href="#n{{ add .Lineno -10 }}">Go to line {{.Lineno}}</a> |
<a href="#bottom">Go to bottom</a>

<pre>
{{ range $index, $line := .Fp }}<span id="n{{$index}}"{{ if eq $index $.Lineno }} style="background-color: yellow;"{{ end }}>{{ rjust $index 4 }}: {{ $line }}</span>
{{ end }}
</pre>
<a name="bottom" href="#">Back to top</a>
</div>

{{ template "end" . }}
{{ template "footer" . }}
{{ end }}
`
