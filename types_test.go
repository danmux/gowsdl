// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gowsdl

import (
	"encoding/xml"
	"fmt"
	"testing"

	"github.com/kr/pretty"
)

func TestSimpleSchema(t *testing.T) {

	ty := &types{
		schemas: []*XSDSchema{},
	}

	data := []byte(simple)
	err := xml.Unmarshal(data, &ty.schemas)
	if err != nil {
		t.Error(err)
	}

	err = xml.Unmarshal(data, &ty.schemaMetas)
	if err != nil {
		t.Error(err)
	}

	if ty.schemaMetas[0].xsdPrefix() != "s" {
		t.Error("should have got xsd prefix \"s\" got", ty.schemaMetas[0].xsdPrefix())
	}

	if ty.schemaMetas[0].tnsPrefix() != "" {
		t.Error("should have got prefix \"\" got", ty.schemaMetas[0].tnsPrefix())
	}

	// fmt.Printf("%#v", ty)

	// b, err := ty.genTypes()
	// if err != nil {
	// 	t.Error(err)
	// }

	// t.Log(string(b))

	t.Log(fmt.Printf("%# v\n\n", pretty.Formatter(ty)))
}

func TestComplexSchema(t *testing.T) {

	ty := &types{
		schemas: []*XSDSchema{},
	}

	data := []byte(complex)
	err := xml.Unmarshal(data, &ty.schemas)
	if err != nil {
		t.Error(err)
	}

	err = xml.Unmarshal(data, &ty.schemaMetas)
	if err != nil {
		t.Error(err)
	}

	if ty.schemaMetas[0].xsdPrefix() != "" {
		t.Error("should have got xsd prefix \"\" got", ty.schemaMetas[0].xsdPrefix())
	}

	if ty.schemaMetas[0].tnsPrefix() != "tns" {
		t.Error("should have got prefix \"tns\" got", ty.schemaMetas[0].tnsPrefix())
	}
	// ty.genTypes()

	t.Log(fmt.Printf("%# v\n\n", pretty.Formatter(ty)))
}

// an element is a named field - which can be a field in a struct or a parameter to a function

var simple = `
    <s:schema elementFormDefault="qualified" targetNamespace="http://www.webserviceX.NET" xmlns:s="http://www.w3.org/2001/XMLSchema">
      <s:element name="GetWeather">
        <s:complexType>
          <s:sequence>
            <s:element minOccurs="0" maxOccurs="1" name="CityName" type="s:string" />
            <s:element minOccurs="0" maxOccurs="1" name="CountryName" type="s:string" />
          </s:sequence>
        </s:complexType>
      </s:element>
      <s:element name="GetWeatherResponse">
        <s:complexType>
          <s:sequence>
            <s:element minOccurs="0" maxOccurs="1" name="GetWeatherResult" type="s:string" />
          </s:sequence>
        </s:complexType>
      </s:element>
      <s:element name="GetCitiesByCountry">
        <s:complexType>
          <s:sequence>
            <s:element minOccurs="0" maxOccurs="1" name="CountryName" type="s:string" />
          </s:sequence>
        </s:complexType>
      </s:element>
      <s:element name="GetCitiesByCountryResponse">
        <s:complexType>
          <s:sequence>
            <s:element minOccurs="0" maxOccurs="1" name="GetCitiesByCountryResult" type="s:string" />
          </s:sequence>
        </s:complexType>
      </s:element>
      <s:element name="string" nillable="true" type="s:string" />
    </s:schema>
`

var complex = `
<schema xmlns:tns="urn:description7a.services.chrome.com" xmlns="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:description7a.services.chrome.com" elementFormDefault="qualified">
    <complexType name="AccountInfo">
        <attribute name="number" type="string" use="required">
            <annotation>
                <documentation>Account Number provided by Chrome.</documentation>
            </annotation>
        </attribute>
        <attribute name="secret" type="string" use="required">
            <annotation>
                <documentation>Account Secret/Password provided by Chrome.</documentation>
            </annotation>
        </attribute>
        <attribute name="country" type="string" use="required">
            <annotation>
                <documentation>Upper-case, two-letter code defined by ISO-3166.</documentation>
            </annotation>
        </attribute>
        <attribute name="language" type="string" use="required">
            <annotation>
                <documentation>Lower-case, two-letter code defined by ISO-639.</documentation>
            </annotation>
        </attribute>
        <attribute name="behalfOf" type="string" />
    </complexType>

    <element name="VersionInfo">
        <complexType>
            <complexContent>
                <extension base="tns:BaseResponse">
                    <sequence>
                        <element name="data" maxOccurs="unbounded">
                            <annotation>
                                <documentation>Represents each country of data available via this service, its
                                    version, and licensed availability.
                                </documentation>
                            </annotation>
                            <complexType>
                                <attribute name="country" type="string" use="required">
                                    <annotation>
                                        <documentation>Upper-case, two-letter country code defined by ISO-3166.
                                        </documentation>
                                    </annotation>
                                </attribute>
                                <attribute name="build" type="string" use="required">
                                    <annotation>
                                        <documentation>The unique version number for this data set.
                                        </documentation>
                                    </annotation>
                                </attribute>
                                <attribute name="date" type="dateTime" use="required">
                                    <annotation>
                                        <documentation>The time at which this data was published.
                                        </documentation>
                                    </annotation>
                                </attribute>
                                <attribute name="licensed" type="boolean">
                                    <annotation>
                                        <documentation>True if these data are licensed.</documentation>
                                    </annotation>
                                </attribute>
                            </complexType>
                        </element>
                    </sequence>
                </extension>
            </complexContent>
        </complexType>
    </element>

    <complexType name="BaseResponse">
        <sequence>
            <element name="responseStatus" type="tns:ResponseStatus" />
        </sequence>
    </complexType>

    <element name="ModelYears">
        <complexType>
            <complexContent>
                <extension base="tns:BaseResponse">
                    <sequence>
                        <element name="modelYear" type="int" minOccurs="0" maxOccurs="unbounded" />
                    </sequence>
                </extension>
            </complexContent>
        </complexType>
    </element>

    <element name="Divisions">
        <complexType>
            <complexContent>
                <extension base="tns:BaseResponse">
                    <sequence>
                        <element name="division" type="tns:IdentifiedString" minOccurs="0" maxOccurs="unbounded" />
                    </sequence>
                </extension>
            </complexContent>
        </complexType>
    </element>

    <element name="Subdivisions">
        <complexType>
            <complexContent>
                <extension base="tns:BaseResponse">
                    <sequence>
                        <element name="subdivision" type="tns:IdentifiedString" minOccurs="0" maxOccurs="unbounded" />
                    </sequence>
                </extension>
            </complexContent>
        </complexType>
    </element>

    <element name="Models">
        <complexType>
            <complexContent>
                <extension base="tns:BaseResponse">
                    <sequence>
                        <element name="model" type="tns:IdentifiedString" minOccurs="0" maxOccurs="unbounded" />
                    </sequence>
                </extension>
            </complexContent>
        </complexType>
    </element>

    <element name="Styles">
        <complexType>
            <complexContent>
                <extension base="tns:BaseResponse">
                    <sequence>
                        <element name="style" type="tns:IdentifiedString" minOccurs="0" maxOccurs="unbounded" />
                    </sequence>
                </extension>
            </complexContent>
        </complexType>
    </element>

    <complexType name="Style">
        <sequence>
            <element name="division" type="tns:IdentifiedString" />
            <element name="subdivision" type="tns:IdentifiedString" />
            <element name="model" type="tns:IdentifiedString" />

            <element name="basePrice" type="tns:Price" minOccurs="0" />
            <element name="bodyType" minOccurs="0" maxOccurs="unbounded">
                <complexType>
                    <complexContent>
                        <extension base="tns:IdentifiedString">
                            <attribute name="primary" type="boolean" />
                        </extension>
                    </complexContent>
                </complexType>
            </element>
            <element name="marketClass" type="tns:IdentifiedString" minOccurs="0" />
            <element name="stockImage" minOccurs="0">
                <complexType>
                    <complexContent>
                        <extension base="tns:Image">
                            <attribute name="filename" type="string" use="required" />
                        </extension>
                    </complexContent>
                </complexType>
            </element>
            <element name="mediaGallery" type="tns:MediaGallery" minOccurs="0" />
        </sequence>
        <attribute name="id" type="int" use="required" />
        <attribute name="modelYear" type="int" use="required" />
        <attribute name="name" type="string" use="required" />
        <attribute name="nameWoTrim" type="string" />
        <attribute name="trim" type="string" />
        <attribute name="mfrModelCode" type="string" />
        <attribute name="fleetOnly" type="boolean" />
        <attribute name="modelFleet" type="boolean" />
        <attribute name="passDoors" type="int" />
        <attribute name="altModelName" type="string" />
        <attribute name="altStyleName" type="string" />
        <attribute name="altBodyType" type="string" />
        <attribute name="drivetrain" type="tns:DriveTrain" />
    </complexType>

    <complexType name="Price">
        <attribute name="unknown" type="boolean" />
        <attribute name="invoice" type="double" />
        <attribute name="msrp" type="double" />
        <attribute name="destination" type="double" />
    </complexType>

    <complexType name="PriceRange">
        <sequence>
            <element name="invoice" type="tns:Range" />
            <element name="msrp" type="tns:Range" />
            <element name="destination" type="tns:Range" />
        </sequence>
        <attribute name="unknown" type="boolean" />
    </complexType>

    <simpleType name="DriveTrain">
        <restriction base="string">
            <enumeration value="Front Wheel Drive" />
            <enumeration value="Rear Wheel Drive" />
            <enumeration value="All Wheel Drive" />
            <enumeration value="Four Wheel Drive" />
        </restriction>
    </simpleType>

    <element name="VehicleDescription">
        <complexType>
            <complexContent>
                <extension base="tns:BaseResponse">
                    <sequence>
                        <element name="vinDescription" minOccurs="0">
                            <complexType>
                                <sequence>
                                    <element name="gvwr" type="tns:Range" minOccurs="0" />
                                    <element name="WorldManufacturerIdentifier" type="string" minOccurs="0" />
                                    <element name="ManufacturerIdentificationCode" type="string" minOccurs="0" />
                                    <element name="restraintTypes" type="tns:CategoryDefinition" minOccurs="0" maxOccurs="unbounded" />
                                    <element name="marketClass" type="tns:IdentifiedString" minOccurs="0" maxOccurs="unbounded" />
                                </sequence>
                                <attribute name="vin" type="string" use="required" />
                                <attribute name="modelYear" type="int" use="required" />
                                <attribute name="division" type="string" use="required" />
                                <attribute name="modelName" type="string" use="required" />
                                <attribute name="styleName" type="string" />
                                <attribute name="bodyType" type="string" />
                                <attribute name="drivingWheels" type="string" />
                                <attribute name="built" type="dateTime" />
                            </complexType>
                        </element>
                        <element name="style" type="tns:Style" minOccurs="0" maxOccurs="unbounded" />
                        <element name="engine" type="tns:Engine" minOccurs="0" maxOccurs="unbounded" />
                        <element name="standard" type="tns:Standard" minOccurs="0" maxOccurs="unbounded" />
                        <element name="factoryOption" type="tns:Option" minOccurs="0" maxOccurs="unbounded" />
                        <element name="genericEquipment" type="tns:GenericEquipment" minOccurs="0" maxOccurs="unbounded" />
                        <element name="consumerInformation" type="tns:ConsumerInformation" minOccurs="0" maxOccurs="unbounded" />
                        <element name="technicalSpecification" type="tns:TechnicalSpecification" minOccurs="0" maxOccurs="unbounded" />
                        <element name="exteriorColor" type="tns:Color" minOccurs="0" maxOccurs="unbounded" />
                        <element name="interiorColor" type="tns:Color" minOccurs="0" maxOccurs="unbounded" />
                        <element name="genericColor" type="tns:GenericColor" minOccurs="0" maxOccurs="unbounded" />
                        <element name="basePrice" type="tns:PriceRange" minOccurs="0" />
                    </sequence>
                    <attribute name="country" type="string" use="required" />
                    <attribute name="language" type="string" use="required" />
                    <attribute name="modelYear" type="int" />
                    <attribute name="bestMakeName" type="string" />
                    <attribute name="bestModelName" type="string" />
                    <attribute name="bestStyleName" type="string" />
                    <attribute name="bestTrimName" type="string" />
                </extension>
            </complexContent>
        </complexType>
    </element>

    <complexType name="Range">
        <attribute name="low" type="double" />
        <attribute name="high" type="double" />
    </complexType>

    <complexType name="InstallationCause">
        <attribute name="cause" use="required">
            <simpleType>
                <restriction base="string">
                    <enumeration value="Engine" />
                    <enumeration value="RelatedCategory" />
                    <enumeration value="RelatedColor" />
                    <enumeration value="CategoryLogic" />
                    <enumeration value="OptionLogic" />
                    <enumeration value="OptionCodeBuild" />
                    <enumeration value="ExteriorColorBuild" />
                    <enumeration value="InteriorColorBuild" />
                    <enumeration value="EquipmentDescriptionInput" />
                    <enumeration value="ExteriorColorInput" />
                    <enumeration value="InteriorColorInput" />
                    <enumeration value="OptionCodeInput" />
                    <enumeration value="BaseEquipment" />
                    <enumeration value="VIN" />
                    <enumeration value="NonFactoryEquipmentInput" />
                </restriction>
            </simpleType>
        </attribute>
        <attribute name="detail" type="string" />
    </complexType>

    <complexType name="Engine">
        <sequence>
            <element name="engineType" type="tns:IdentifiedString" />
            <element name="fuelType" type="tns:IdentifiedString" />
            <element name="horsepower" type="tns:ValueRPM" minOccurs="0" />
            <element name="netTorque" type="tns:ValueRPM" minOccurs="0" />
            <element name="cylinders" type="int" minOccurs="0" />
            <element name="displacement" minOccurs="0">
                <complexType>
                    <attribute name="liters" type="double" />
                    <attribute name="cubicIn" type="int" />
                </complexType>
            </element>
            <element name="fuelEconomy" minOccurs="0">
                <complexType>
                    <sequence>
                        <element name="city" type="tns:Range" />
                        <element name="hwy" type="tns:Range" />
                    </sequence>
                    <attribute name="unit" type="string" use="required" />
                </complexType>
            </element>
            <element name="fuelCapacity" minOccurs="0">
                <complexType>
                    <complexContent>
                        <extension base="tns:Range">
                            <attribute name="unit" type="string" use="required" />
                        </extension>
                    </complexContent>
                </complexType>
            </element>
            <element name="forcedInduction" type="tns:IdentifiedString" minOccurs="0" />
            <element name="installed" type="tns:InstallationCause" minOccurs="0" />
        </sequence>
        <attribute name="highOutput" type="boolean" />
    </complexType>

    <complexType name="ValueRPM">
        <attribute name="value" type="double" />
        <attribute name="rpm" type="int" />
    </complexType>

    <complexType name="Standard">
        <sequence>
            <element name="header" type="tns:IdentifiedString" />
            <element name="description" type="string" />
            <element name="category" type="tns:CategoryAssociation" minOccurs="0" maxOccurs="unbounded" />
            <element name="styleId" type="int" maxOccurs="unbounded" />
            <element name="installed" type="tns:InstallationCause" minOccurs="0" />
        </sequence>
    </complexType>

    <complexType name="CategoryAssociation">
        <attribute name="id" type="int" use="required" />
        <attribute name="removed" type="boolean" />
    </complexType>

    <complexType name="Option">
        <sequence>
            <element name="header" type="tns:IdentifiedString" minOccurs="0" />
            <element name="description" type="string" minOccurs="0" maxOccurs="unbounded" />
            <element name="category" type="tns:CategoryAssociation" minOccurs="0" maxOccurs="unbounded" />

            <element name="price" type="tns:OptionPrice" minOccurs="0" />
            <element name="styleId" type="int" minOccurs="0" maxOccurs="unbounded" />
            <element name="installed" type="tns:InstallationCause" minOccurs="0" />
            <element name="ambiguousOption" type="tns:Option" minOccurs="0" maxOccurs="unbounded" />
        </sequence>
        <attribute name="chromeCode" type="string" />
        <attribute name="oemCode" type="string" />
        <attribute name="altOptionCode" type="string" />
        <attribute name="standard" type="boolean" />
        <attribute name="optionKindId" type="int" />
        <attribute name="utf" type="string" />
        <attribute name="fleetOnly" type="boolean" />
    </complexType>

    <complexType name="OptionPrice">
        <attribute name="unknown" type="boolean" />
        <attribute name="invoiceMin" type="double" />
        <attribute name="invoiceMax" type="double" />
        <attribute name="msrpMin" type="double" />
        <attribute name="msrpMax" type="double" />
    </complexType>

    <complexType name="GenericEquipment">
        <sequence>
            <choice>
                <element name="categoryId" type="int" />
                <element name="definition" type="tns:CategoryDefinition" />
            </choice>
            <sequence>
                <element name="styleId" type="int" minOccurs="0" maxOccurs="unbounded" />
                <element name="installed" type="tns:InstallationCause" minOccurs="0" />
            </sequence>
        </sequence>
    </complexType>

    <complexType name="ConsumerInformation">
        <sequence>
            <element name="type" type="tns:IdentifiedString" />
            <element name="item" minOccurs="0" maxOccurs="unbounded">
                <complexType>
                    <attribute name="name" type="string" use="required" />
                    <attribute name="conditionNote" type="string" />
                    <attribute name="value" type="string" />
                </complexType>
            </element>
            <element name="styleId" type="int" maxOccurs="unbounded" />
        </sequence>
    </complexType>

    <complexType name="TechnicalSpecification">
        <sequence>
            <choice>
                <element name="titleId" type="int" />
                <element name="definition" type="tns:TechnicalSpecificationDefinition" />
            </choice>
            <sequence>
                <element name="range" minOccurs="0">
                    <complexType>
                        <attribute name="min" type="double" use="required" />
                        <attribute name="max" type="double" use="required" />
                    </complexType>
                </element>
                <element name="value" minOccurs="0" maxOccurs="unbounded">
                    <complexType>
                        <sequence>
                            <element name="styleId" type="int" maxOccurs="unbounded" />
                        </sequence>
                        <attribute name="value" type="string" />
                        <attribute name="condition" type="string" />
                    </complexType>
                </element>
            </sequence>
        </sequence>
    </complexType>

    <complexType name="GenericColor">
        <sequence>
            <element name="installed" type="tns:InstallationCause" minOccurs="0" />
        </sequence>
        <attribute name="name" type="string" use="required" />
        <attribute name="primary" type="boolean" use="optional" />
    </complexType>

    <complexType name="Color">
        <sequence>
            <element name="genericColor" type="tns:GenericColor" minOccurs="0" maxOccurs="unbounded" />
            <element name="styleId" type="int" maxOccurs="unbounded" />
            <element name="installed" type="tns:InstallationCause" minOccurs="0" />
        </sequence>
        <attribute name="colorCode" type="string" use="required" />
        <attribute name="colorName" type="string" use="required" />
        <attribute name="rgbValue" type="string" />
    </complexType>

    <complexType name="ResponseStatus">
        <sequence>
            <element name="matchedEquipment" type="tns:MatchedEquipment" minOccurs="0" maxOccurs="unbounded" />
            <element name="matchedNonFactoryEquipment" type="tns:MatchedNonFactoryEquipment" minOccurs="0" maxOccurs="unbounded" />
            <element name="status" minOccurs="0" maxOccurs="unbounded">
                <complexType>
                    <simpleContent>
                        <extension base="string">
                            <attribute name="code" use="required">
                                <simpleType>
                                    <restriction base="string">
                                        <enumeration value="UnrecognizedTrimName" />
                                        <enumeration value="UnusedTrimName" />
                                        <enumeration value="UnrecognizedManufacturerModelCode" />
                                        <enumeration value="UnusedManufacturerModelCode" />
                                        <enumeration value="UnrecognizedStyleId" />
                                        <enumeration value="UnusedReducingStyleId" />
                                        <enumeration value="UnrecognizedReducingStyleId" />
                                        <enumeration value="UnrecognizedWheelBase" />
                                        <enumeration value="UnusedWheelBase" />
                                        <enumeration value="UnrecognizedOptionCode" />
                                        <enumeration value="UnusedOptionCode" />
                                        <enumeration value="UnrecognizedEquipmentDescription" />
                                        <enumeration value="UnrecognizedNonFactoryEquipmentDescription" />
                                        <enumeration value="UnusedNonFactoryEquipmentDescription" />
                                        <enumeration value="UsingAlternateLocale" />
                                        <enumeration value="NameMatchNotFound" />
                                        <enumeration value="VinNotCarriedByChrome" />
                                        <enumeration value="InvalidVinCheckDigit" />
                                        <enumeration value="InvalidVinCharacter" />
                                        <enumeration value="InvalidVinLength" />
                                        <enumeration value="UnrecognizedInteriorColor" />
                                        <enumeration value="UnrecognizedExteriorColor" />
                                        <enumeration value="UnrecognizedTechnicalSpecificationTitleId" />
                                        <enumeration value="Unexpected" />
                                    </restriction>
                                </simpleType>
                            </attribute>
                        </extension>
                    </simpleContent>
                </complexType>
            </element>
        </sequence>
        <attribute name="responseCode" use="required">
            <simpleType>
                <restriction base="string">
                    <enumeration value="Successful" />
                    <enumeration value="Unsuccessful" />
                    <enumeration value="ConditionallySuccessful" />
                </restriction>
            </simpleType>
        </attribute>
        <attribute name="description" type="string" />
    </complexType>

    <complexType name="MatchedEquipment">
        <sequence>
            <element name="equipmentDescription" type="string" />
            <element name="categoryId" type="int" maxOccurs="unbounded" />
        </sequence>
    </complexType>
    <complexType name="MatchedNonFactoryEquipment">
        <sequence>
            <element name="equipmentDescription" type="string" />
            <element name="category" type="tns:CategoryDefinition" maxOccurs="unbounded" />
            <element name="installed" type="tns:InstallationCause" minOccurs="0" />
        </sequence>
    </complexType>

    <complexType name="IdentifiedString">
        <simpleContent>
            <extension base="string">
                <attribute name="id" type="int" use="required" />
            </extension>
        </simpleContent>
    </complexType>

    <complexType name="CategoryDefinition">
        <sequence>
            <element name="group" type="tns:IdentifiedString" />
            <element name="header" type="tns:IdentifiedString" />
            <element name="category" type="tns:IdentifiedString" />
            <element name="type" type="tns:IdentifiedString" minOccurs="0" />
        </sequence>
    </complexType>

    <element name="CategoryDefinitions">
        <complexType>
            <complexContent>
                <extension base="tns:BaseResponse">
                    <sequence>
                        <element name="category" type="tns:CategoryDefinition" minOccurs="0" maxOccurs="unbounded" />
                    </sequence>
                </extension>
            </complexContent>
        </complexType>
    </element>

    <complexType name="TechnicalSpecificationDefinition">
        <sequence>
            <element name="group" type="tns:IdentifiedString" />
            <element name="header" type="tns:IdentifiedString" />
            <element name="title" type="tns:IdentifiedString" />
        </sequence>
        <attribute name="measurementUnit" type="string" />
    </complexType>

    <element name="TechnicalSpecificationDefinitions">
        <complexType>
            <complexContent>
                <extension base="tns:BaseResponse">
                    <sequence>
                        <element name="definition" type="tns:TechnicalSpecificationDefinition" minOccurs="0" maxOccurs="unbounded" />
                    </sequence>
                </extension>
            </complexContent>
        </complexType>
    </element>

    <complexType name="MediaGallery">
        <sequence>
            <element name="view" minOccurs="0" maxOccurs="unbounded">
                <complexType>
                    <complexContent>
                        <extension base="tns:Image">
                            <attribute name="shotCode" type="string" />
                            <attribute name="backgroundDescription" type="string" use="optional" />
                        </extension>
                    </complexContent>
                </complexType>
            </element>
            <element name="colorized" minOccurs="0" maxOccurs="unbounded">
                <complexType>
                    <complexContent>
                        <extension base="tns:Image">
                            <attribute name="primaryColorOptionCode" type="string" use="required" />
                            <attribute name="secondaryColorOptionCode" type="string" use="optional" />
                            <attribute name="match" type="boolean" use="optional" />
                            <attribute name="shotCode" type="string" />
                            <attribute name="backgroundDescription" type="string" use="optional" />
                            <attribute name="primaryRGBHexCode" type="string" use="optional" />
                            <attribute name="secondaryRGBHexCode" type="string" use="optional" />
                        </extension>
                    </complexContent>
                </complexType>
            </element>
        </sequence>
        <attribute name="styleId" type="int" />
    </complexType>

    <complexType name="Image">
        <attribute name="url" type="string" use="required" />
        <attribute name="width" type="int" />
        <attribute name="height" type="int" />
    </complexType>

    <!-- /// Requests /// -->

    <complexType name="BaseRequest">
        <sequence>
            <element name="accountInfo" type="tns:AccountInfo" />
        </sequence>
    </complexType>

    <element name="VersionInfoRequest" type="tns:BaseRequest" />

    <element name="ModelYearsRequest" type="tns:BaseRequest" />

    <element name="DivisionsRequest">
        <complexType>
            <annotation>
                <documentation>Provides a list of Chrome division ID's associated with the provided year.
                </documentation>
            </annotation>
            <complexContent>
                <extension base="tns:BaseRequest">
                    <attribute name="modelYear" type="int" use="required" />
                </extension>
            </complexContent>
        </complexType>
    </element>

    <element name="SubdivisionsRequest">
        <annotation>
            <documentation>Provides a list of Chrome subdivision ID's associated with the provided year.
            </documentation>
        </annotation>
        <complexType>
            <complexContent>
                <extension base="tns:BaseRequest">
                    <attribute name="modelYear" type="int" use="required" />
                </extension>
            </complexContent>
        </complexType>
    </element>

    <element name="ModelsRequest">
        <annotation>
            <documentation>Provides a list of Chrome model ID's associated with the provided year and
                (sub)division ID.
            </documentation>
        </annotation>
        <complexType>
            <complexContent>
                <extension base="tns:BaseRequest">
                    <sequence>
                        <element name="modelYear" type="int" />
                        <choice>
                            <element name="divisionId" type="int" />
                            <element name="subdivisionId" type="int" />
                        </choice>
                    </sequence>
                </extension>
            </complexContent>
        </complexType>
    </element>

    <element name="StylesRequest">
        <annotation>
            <documentation>Provides a list of Chrome style ID's associated with the provided model ID.
            </documentation>
        </annotation>
        <complexType>
            <complexContent>
                <extension base="tns:BaseRequest">
                    <attribute name="modelId" type="int" use="required" />
                </extension>
            </complexContent>
        </complexType>
    </element>

    <simpleType name="Switch">
        <annotation>
            <documentation>Adding one or more switch strings to your request will change the behavior of ADS.
                Use the following switches to match output with your particular needs.
            </documentation>
        </annotation>
        <restriction base="string">
            <enumeration value="DisableSafeStandards">
                <annotation>
                    <documentation>By default, only equipment that could not have been upgraded or removed will
                        be presented as installed. When you use this switch, any equipment that could
                        be standard equipment will be installed even if they could have been removed
                        or upgraded.
                    </documentation>
                </annotation>
            </enumeration>
            <enumeration value="ShowExtendedDescriptions">
                <annotation>
                    <documentation>Causes ADS to provide additional description information for each piece of
                        equipment.
                    </documentation>
                </annotation>
            </enumeration>
            <enumeration value="ShowAvailableEquipment">
                <annotation>
                    <documentation>Causes ADS to show information about all equipment available for the vehicle,
                        whether or not it is installed.
                    </documentation>
                </annotation>
            </enumeration>
            <enumeration value="ShowConsumerInformation">
                <annotation>
                    <documentation>Causes ADS to show normalized consumer information such as recalls, awards,
                        and test results.
                    </documentation>
                </annotation>
            </enumeration>
            <enumeration value="ShowExtendedTechnicalSpecifications">
                <annotation>
                    <documentation>Causes ADS to show all available technical specifications for the vehicle,
                        and additional information about them.
                    </documentation>
                </annotation>
            </enumeration>
            <enumeration value="IncludeRegionalVehicles">
                <annotation>
                    <documentation>By default, only vehicles sold nationally are considered for description.
                        This switch causes ADS to also consider vehicles sold only regionally.
                    </documentation>
                </annotation>
            </enumeration>
            <enumeration value="UseDependencyOrderingLogic">
                <annotation>
                    <documentation>By default, ADS describes and installs only equipment specifically
                        known to exist (usually because of user input.) This switch causes ADS
                        to consider ordering logic caused by the installed equipment itself
                        in addition to the ordering logic of the user-identified equipment.
                    </documentation>
                </annotation>
            </enumeration>
            <enumeration value="IncludeDefinitions">
                <annotation>
                    <documentation>Causes ADS to show Category and Technical Specification definitions in-line within a vehicle description.</documentation>
                </annotation>
            </enumeration>
        </restriction>
    </simpleType>

    <simpleType name="SwitchAvailability">

        <restriction base="string">
            <enumeration value="ExcludeFleetOnly">
                <annotation>
                    <documentation>Excludes Fleet Only information. (Default is "both.")</documentation>
                </annotation>
            </enumeration>
            <enumeration value="ExcludeRetailOnly">
                <annotation>
                    <documentation>Excludes Retail Only information. (Default is "both".)</documentation>
                </annotation>
            </enumeration>
        </restriction>
    </simpleType>

    <simpleType name="SwitchChromeMediaGallery">
        <annotation>
            <documentation>Provides a Chrome Media Gallery URL's associated with the described vehicle.
                Your user license dictates which views (none, multi-view, colorMatch, or both) are
                available. The default value is the most your license permits (hopefully "both.")
            </documentation>
        </annotation>
        <restriction base="string">
            <enumeration value="Multi-View">
                <annotation>
                    <documentation>Provide Multi-view images, if the client license permits.</documentation>
                </annotation>
            </enumeration>
            <enumeration value="ColorMatch">
                <annotation>
                    <documentation>Provide ColorMatch images, if the client license permits.</documentation>
                </annotation>
            </enumeration>
            <enumeration value="Both">
                <annotation>
                    <documentation>Provide both image types, if the client license permits.</documentation>
                </annotation>
            </enumeration>
        </restriction>
    </simpleType>

    <element name="VehicleDescriptionRequest">
        <annotation>
            <documentation>
                Describe a vehicle. You must provide one of: vehicle identifier
                (VIN or HIN); Chrome style ID; or year, make name, and model name. Optional input fields can
                help identification by limiting color, trim, wheelbase, and installed options.
            </documentation>
        </annotation>
        <complexType>
            <complexContent>
                <extension base="tns:BaseRequest">
                    <sequence>
                        <choice>
                            <sequence>
                                <annotation>
                                    <documentation>
                                        Causes ADS to describe a vehicle from a given year, make, and model. All
                                        three must be populated.
                                    </documentation>
                                </annotation>
                                <element name="modelYear" type="int" />
                                <element name="makeName" type="string" />
                                <element name="modelName" type="string" />
                            </sequence>
                            <sequence>
                                <annotation>
                                    <documentation>
                                        Causes ADS to describe a vehicle from a given vehicle identifier (VIN
                                        or HIN.) You can optionally provide a known Chrome style ID to help
                                        the identification process.
                                    </documentation>
                                </annotation>
                                <element name="vin" type="string" />
                                <element name="reducingStyleId" type="int" minOccurs="0" />
                            </sequence>
                            <element name="styleId" type="int">
                                <annotation>
                                    <documentation>Causes ADS to find and describe the given vehicle.
                                    </documentation>
                                </annotation>
                            </element>
                        </choice>
                        <element name="trimName" type="string" minOccurs="0">
                            <annotation>
                                <documentation>Trim names are typically things like "XLT", "Sport" or "Eddie
                                    Bauer".
                                </documentation>
                            </annotation>
                        </element>
                        <element name="manufacturerModelCode" type="string" minOccurs="0">
                            <annotation>
                                <documentation>MMC are typically things like "TK10743"or "CC10706".
                                </documentation>
                            </annotation>
                        </element>
                        <element name="wheelBase" type="double" minOccurs="0">
                            <annotation>
                                <documentation>
                                    Give wheel base in inches. ADS will try to find vehicles where (1) the
                                    wheel base matters in the identification (usually Ford pickups) and (2)
                                    within +/- 2" of the given value. Round to the nearest whole inch. If
                                    you don't, ADS will.
                                </documentation>
                            </annotation>
                        </element>
                        <element name="OEMOptionCode" type="string" minOccurs="0" maxOccurs="unbounded">
                            <annotation>
                                <documentation>
                                    OEM option codes are identifiers that manufacturers use to
                                    identify which options and packages to install on a specific vehicle. The
                                    codes to use are unique to each manufacturer and will look like "FF3" or
                                    "AJX". You can provide as many of these as you know, but only one per
                                    element.
                                </documentation>
                            </annotation>
                        </element>
                        <element name="equipmentDescription" type="string" minOccurs="0" maxOccurs="unbounded">
                            <annotation>
                                <documentation>
                                    Provide the name and or description of equipment you know to be installed.
                                    If you know the manufacturer's actual name use it. Otherwise use the most
                                    descriptive name you can think of. You can provide as many of these as
                                    you know, but only one per element.
                                </documentation>
                            </annotation>
                        </element>
                        <element name="exteriorColorName" type="string" minOccurs="0">
                            <annotation>
                                <documentation>
                                    The name of the exterior color. If you know the manufacturer's actual
                                    color name, use it. Otherwise use the most reasonable color you can
                                    think of.
                                </documentation>
                            </annotation>
                        </element>
                        <element name="interiorColorName" type="string" minOccurs="0">
                            <annotation>
                                <documentation>
                                    The name of the interior color or interior color pair. If you know the
                                    manufacturer's actual color name, use it. Otherwise use the most
                                    reasonable color you can think of.
                                </documentation>
                            </annotation>
                        </element>
                        <element name="nonFactoryEquipmentDescription" type="string" minOccurs="0" maxOccurs="unbounded">
                            <annotation>
                                <documentation>
                                    Provide the name and or description of non-factory (aftermarket) equipment
                                    you know to be installed. This equipment will be listed as installed
                                    non-factory equipment, without validation against manufacturer's install
                                    logic and will not affect the identification or installation of factory
                                    options, packages or equipment. You can provide as many of these as
                                    you know, but only one per element.
                                </documentation>
                            </annotation>
                        </element>

                        <!-- return parameters -->
                        <element name="switch" type="tns:Switch" minOccurs="0" maxOccurs="unbounded" />
                        <element name="vehicleProcessMode" type="tns:SwitchAvailability" minOccurs="0">
                            <annotation>
                                <documentation>The default behavior of ADS is to include both fleet-only and
                                    retail only styles when discovering vehicles. Use this switch to tell
                                    ADS to ignore either or both.
                                </documentation>
                            </annotation>
                        </element>
                        <element name="optionsProcessMode" type="tns:SwitchAvailability" minOccurs="0">
                            <annotation>
                                <documentation>The default behavior of ADS is to include both fleet-only and
                                    retail only options when discovering equipment. Use this switch to tell
                                    ADS to ignore either or both.
                                </documentation>
                            </annotation>
                        </element>
                        <element name="includeMediaGallery" type="tns:SwitchChromeMediaGallery" minOccurs="0">
                            <annotation>
                                <documentation>
                                    If your license allows, ADS will provide additional images (beyond the
                                    stock image) for each style described in the output. Chrome Media gallery
                                    supports "colorMatch" (where the image is the designated color), "multiView"
                                    (where the vehicle is seen from several angles) and "both." See the
                                    documentation for the switch type for specific instructions.
                                </documentation>
                            </annotation>
                        </element>
                        <element name="includeTechnicalSpecificationTitleId" type="int" minOccurs="0" maxOccurs="unbounded">
                            <annotation>
                                <documentation>The default behavior of ADS is to include all available technical specifications.
                                    Use this switch to tell ADS specific technical specifications (by title id) to be shown.
                                </documentation>
                            </annotation>
                        </element>
                    </sequence>
                </extension>
            </complexContent>
        </complexType>
    </element>

    <element name="CategoryDefinitionsRequest" type="tns:BaseRequest">
        <annotation>
            <documentation>
                Provide a list of all available equipment category ID's.
            </documentation>
        </annotation>
    </element>

    <element name="TechnicalSpecificationDefinitionsRequest" type="tns:BaseRequest">
        <annotation>
            <documentation>
                Provide a list of all available technical specification definitions.
            </documentation>
        </annotation>
    </element>
</schema>
  
    `
