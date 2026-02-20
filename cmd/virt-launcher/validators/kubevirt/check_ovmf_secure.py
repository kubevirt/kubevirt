loader = vl.dom_xpath("//domain/os/loader[1]/text()") or ""
secure = vl.dom_xpath("//domain/os/loader[1]/@secure") or ""

if "secboot" in loader and secure != "yes":
    vl.add_warning(vl.WarningDomain_Domain, vl.WarningLevel_Notice,
                   "Secure OVMF code used without secure='yes'")
