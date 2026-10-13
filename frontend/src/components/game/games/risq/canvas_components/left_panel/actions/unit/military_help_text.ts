export const MILITARY_HELP_TEXT = `Military unit behavior

Aggressive: Automatically finds and pursues enemies within one space. Ranged units also attack enemies anywhere within their weapon range, such as two spaces for a range-2 unit.

Defensive: Melee units can move within their current space to attack. Ranged units automatically attack enemies already within range.

Stand Ground: Automatically attacks enemies already within range without pursuing them. Explicit movement orders still work.

Passive: Does not automatically seek enemies. Attack Back can still trigger retaliation within attack range.

Interrupt Current: Allows automatic attacks to replace existing orders. Automatic searches keep explicitly assigned unit or building attack targets, but Attack Back can replace them when this toggle is enabled.

Attack Back: Responds to reachable attackers from the latest recorded attack tick. Without Interrupt Current, Passive retaliates only while idle; other stances can interrupt non-attack orders but keep existing attacks. Every stance permits retaliation within attack range. Aggressive also permits pursuit within one space; Defensive also permits pursuit within the current space.

Target priority: Click an icon to add its category, drag ranked icons to reorder, and click a ranked icon to remove it. Ranked categories are tried first; unranked categories remain fallback targets. Nearer targets win within a category. Priority applies to automatic attacks, retaliation, and attack-space or attack-zone orders.

Multiple units: A solid highlight means all selected units share that setting. Stripes mark stances present in a mixed selection or a boolean with both on and off values. Click a stance to apply it to everyone. Left-click a striped boolean to enable it for everyone; right-click to disable it. Different priority lists show a striped Mixed container. Click a category to give everyone that shared priority, then add, remove, or reorder categories normally.`;
