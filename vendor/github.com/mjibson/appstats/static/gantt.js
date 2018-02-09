// Copyright 2009 Google Inc. All Rights Reserved.

/**
 * Defines a class that can render a simple Gantt chart.
 *
 * @author guido@google.com (Guido van Rossum)
 * @author schefflerjens@google.com (Jens Scheffler)
 */

/**
 * @constructor
 */
var Gantt = function() {
  /**
   * @type {Array}
   */
  this.bars = [];

  /**
   * @type {Array}
   */
  this.output = [];
};


/**
 * Internal fields used to render the chart.
 * Should not be modified.
 * @type {Array.<Array>}
 */
Gantt.SCALES = [[5, 0.2, 1.0],
                [6, 0.2, 1.2],
                [5, 0.25, 1.25],
                [6, 0.25, 1.5],
                [4, 0.5, 2.0],
                [5, 0.5, 2.5],
                [6, 0.5, 3.0],
                [4, 1.0, 4.0],
                [5, 1.0, 5.0],
                [6, 1.0, 6.0],
                [4, 2.0, 8.0],
                [5, 2.0, 10.0]];


/**
 * Helper to compute the proper X axis scale.
 * Args:
 *     highest: the highest value in the data series.
 *
 * Returns:
 *  A tuple (howmany, spacing, limit) where howmany is the number of
 *  increments, spacing is the increment to be used between successive
 *  axis labels, and limit is the rounded-up highest value of the
 *  axis.  Within float precision, howmany * spacing == highest will
 *  hold.
 *
 * The axis is assumed to always start at zero.
 */
Gantt.compute_scale = function(highest) {
  if (highest <= 0) {
    return [2, 0.5, 1.0]  // Special-case if there's no data.
  }
  var scale = 1.0
  while (highest < 1.0) {
    highest *= 10.0
    scale /= 10.0
  }
  while (highest >= 10.0) {
    highest /= 10.0
    scale *= 10.0
  }
  // Now 1 <= highest < 10
  for (var i = 0; i < Gantt.SCALES.length; i++) {
    if (highest <= Gantt.SCALES[i][2]) {
      return [Gantt.SCALES[i][0], Gantt.SCALES[i][1] * scale,
          Gantt.SCALES[i][2] * scale];
    }
  }
  // Avoid the need for "assert False".  Not actually reachable.
  return [5, 2.0 * scale, 10.0 * scale];
};


/**
 * URL of a transparent 1x1 GIF.
 * @type {string}
 */
Gantt.prototype.PIX = 'stats/static/pix.gif';


/**
 * CSS class name prefix.
 * @type {string}
 */
Gantt.prototype.PREFIX = 'ae-stats-gantt-';


/**
 * Height of one bar.
 * @type {string}
 */
Gantt.prototype.HEIGHT = '1em';


/**
 * Height of the extra bar.
 * @type {string}
 */
Gantt.prototype.EXTRA_HEIGHT = '0.5em';


/**
 * Background color for the bar.
 * @type {string}
 */
Gantt.prototype.BG_COLOR = '#eeeeff';


/**
 * Color of the main bar.
 * @type {string}
 */
Gantt.prototype.COLOR = '#7777ff';


/**
 * Color of the extra bar.
 * @type {string}
 */
Gantt.prototype.EXTRA_COLOR = '#ff6666';


/**
 * Font size of inline_label.
 * @type {string}
 */
Gantt.prototype.INLINE_FONT_SIZE = '80%';


/**
 * Top of inline label text.
 * @type {string}
 */
Gantt.prototype.INLINE_TOP = '0.1em';


/**
 * Color for ticks.
 * @type {string}
 */
Gantt.prototype.TICK_COLOR = 'grey';


/**
 * @type {number}
 */
Gantt.prototype.highest_duration = 0;


/*
 * Appends text to the output array.
 * @param {string} text The text to append to the output.
 */
Gantt.prototype.write = function(text) {
  this.output.push(text);
};


/*
 * Internal helper to draw a table row showing the scale.
 * @param {number} howmany
 * @param {number} spacing
 * @param {number} scale
 */
Gantt.prototype.draw_scale = function(howmany, spacing, scale) {
  this.write('<tr class="' + this.PREFIX + 'axisrow">' +
      '<td width="20%"></td><td>');
  this.write('<div class="' + this.PREFIX + 'axis">');
  for (var i = 0; i <= howmany; i++) {
    this.write('<img class="' + this.PREFIX + 'tick" src="' +
          this.PIX + '" alt="" ');
    this.write('style="left:' + (i * spacing * scale) + '%"\n>');
    this.write('<span class="' + this.PREFIX + 'scale" style="left:' +
         (i * spacing * scale) + '%">');
    this.write('&nbsp;' + (i * spacing) + '</span>'); // TODO: number format %4g
  }
  this.write('</div></td></tr>\n');
};


/**
 * Draw the bar chart as HTML.
 */
Gantt.prototype.draw = function() {
  this.output = [];
  var scale = Gantt.compute_scale(this.highest_duration);
  var howmany = scale[0];
  var spacing = scale[1];
  var limit = scale[2];
  scale = 100.0 / limit;
  this.write('<table class="' + this.PREFIX + 'table">\n');
  this.draw_scale(howmany, spacing, scale);
  for (var i = 0; i < this.bars.length; i++) {
    var bar = this.bars[i];
    this.write('<tr class="' + this.PREFIX + 'datarow"><td width="20%">');
    if (bar.label.length > 0) {
      if (bar.link_target.length > 0) {
        this.write('<a class="' + this.PREFIX + 'link" href="' +
              bar.link_target + '">');
      }
      this.write(bar.label);
      if (bar.link_target.length > 0) {
        this.write('</a>');
      }
    }
    this.write('</td>\n<td>');
    this.write('<div class="' + this.PREFIX + 'container">');
    if (bar.link_target.length > 0) {
      this.write('<a class="' + this.PREFIX + 'link" href="' +
            bar.link_target + '"\n>');
    }
    this.write('<img class="' + this.PREFIX + 'bar" src="' +
          this.PIX + '" alt="" ');
    this.write('style="left:' + (bar.start * scale) + '%;width:' +
          (bar.duration * scale) + '%;min-width:1px"\n>');
    if (bar.extra_duration > 0) {
      this.write('<img class="' + this.PREFIX + 'extra" src="' +
            this.PIX + '" alt="" ');
      this.write('style="left:' + (bar.start * scale) + '%;width:' +
            (bar.extra_duration * scale) + '%"\n>');
    }
    if (bar.inline_label.length > 0) {
      this.write('<span class="' + this.PREFIX + 'inline" style="left:' +
            ((bar.start +
              Math.max(bar.duration, bar.extra_duration)) * scale) +
            '%">&nbsp;');
      this.write(bar.inline_label);
      this.write('</span>');
    }
    if (bar.link_target.length > 0) {
      this.write('</a>');
    }
    this.write('</div></td></tr>\n');

  }
  this.draw_scale(howmany, spacing, scale);
  this.write('</table>\n');

  var html = this.output.join('');
  return html;
};


/**
 * Add a bar to the chart.
 * All arguments representing times or durations should be integers
 * or floats expressed in seconds.  The scale drawn is always
 * expressed in seconds (with limited precision).
 * @param {string} label Valid HTML or HTML-escaped text for the left column.
 * @param {number} start Start time for the event.
 * @param {number} duration Duration for the event.
 * @param {number} extra_duration Duration for the second bar; use 0 to
 *     suppress.
 * @param {string} inline_label Valid HTML or HTML-escaped text drawn after the
 *     bars; use '' to suppress.
 * @param {string} link_target HTML-escaped link where clicking on any element
 *       will take you; use '' for no linking.
 */
Gantt.prototype.add_bar = function(label, start, duration, extra_duration,
    inline_label, link_target) {
  this.highest_duration = Math.max(
      this.highest_duration, Math.max(start + duration,
          start + extra_duration));
  this.bars.push({label: label, start: start, duration: duration,
                  extra_duration: extra_duration, inline_label: inline_label,
                  link_target: link_target});
};


goog.exportSymbol('Gantt', Gantt);
goog.exportProperty(Gantt.prototype, 'add_bar', Gantt.prototype.add_bar);
goog.exportProperty(Gantt.prototype, 'draw', Gantt.prototype.draw);
