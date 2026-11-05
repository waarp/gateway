TRANSCODE
=========

Le traitement ``TRANSCODE`` change l'encodage textuel du fichier cible du transfert.
Les arguments sont :

* ``fromCharset`` (*string*) - L'encodage actuel (source) du fichier.
* ``toCharset`` (*string*) - L'encodage souhaité (destination) du fichier.
* ``substitute`` (*bool*) - Écrit le caractère de substitution de l'encodage
  destination à la place d'un caractère qu'il ne sait pas représenter, au lieu
  d'échouer.
* ``records`` (*string*) - Le découpage du fichier source : ``none`` (par
  défaut) le fichier entier, ``lines`` des lignes, ``fixed`` des enregistrements
  de longueur fixe, ``rdw`` des enregistrements précédés de leur longueur (RDW),
  comme z/OS les écrit.
* ``recordLength`` (*int*) - Avec ``fixed`` : la longueur des enregistrements,
  en octets, de 1 à 32760.
* ``trimPadding`` (*bool*) - Avec ``fixed`` : retire les espaces qui terminent
  chaque enregistrement.
* ``outputRecords`` (*string*) - La mise en forme du résultat, quand le fichier
  source est découpé : ``same`` (par défaut) comme le fichier source, ``none``
  sans rien entre les enregistrements, ``lines``, ``fixed`` ou ``rdw``.
* ``outputSeparator`` (*string*) - Avec ``lines`` : la fin de ligne écrite,
  ``LF`` (par défaut), ``CRLF`` ou ``NEL`` (fichier EBCDIC seulement).
* ``outputLength`` (*int*) - Avec ``fixed`` : la longueur des enregistrements
  du résultat, en octets, de 1 à 32760 ; un enregistrement plus court est
  complété par des espaces.
* ``truncate`` (*bool*) - Avec ``fixed`` ou ``rdw`` : coupe un enregistrement
  trop long au lieu d'échouer.

Sans ``records``, le fichier est converti en entier : chaque caractère est lu
dans l'encodage source, puis écrit dans l'encodage destination. Un caractère
que l'encodage destination ne sait pas représenter fait échouer le traitement,
et le fichier reste tel qu'il était.

Avec ``records``, seul le contenu des enregistrements est converti. Une
combinaison d'arguments qui ne s'applique pas (``recordLength`` sans
``fixed``, ``outputSeparator`` sans ``lines``…) est refusée à l'enregistrement
de la règle.

Les fins de ligne sont des octets du fichier. Côté ASCII, ce sont LF et CR LF.
Côté EBCDIC, LF et CR LF sont aussi les octets ``0x0A`` et ``0x0D 0x0A`` : c'est
là que le module PeSIT de la passerelle découpe un fichier texte en articles,
quel que soit son encodage. Un fichier EBCDIC source a de plus ses lignes
reconnues sur ses propres fins de ligne, ``0x25`` et ``0x15`` ; ``NEL`` écrit
``0x15``. Avec ``same``, chaque ligne garde sa fin de ligne octet pour octet :
un fichier converti d'ASCII en EBCDIC par lignes garde ses ``0x0A``, et donc
ses articles quand il part en PeSIT.

.. code-block:: yaml

   pre:
     - type: TRANSCODE
       args:
         fromCharset: "ISO 8859-1"
         toCharset: "IBM Code Page 297"
         records: "lines"

Un encodage se désigne par son nom, tel que les tableaux ci-dessous le donnent,
ou par un nom usuel : la casse, les espaces, les tirets et les soulignés n'y
comptent pas (``utf8`` désigne ``UTF-8``, ``iso-8859-15`` désigne
``ISO 8859-15``), et une page IBM se désigne aussi par son numéro (``IBM1147``,
``CP037``). Deux noms du même encodage sont refusés comme identiques.

Dans l'interface web, le formulaire demande d'abord le sens de la conversion.
ASCII désigne les encodages des systèmes ouverts, UTF-8 compris, et EBCDIC les
pages de codes des systèmes IBM (z/OS, IBM i). Les listes ne proposent ensuite
que les encodages de chaque côté. Le sens n'est pas enregistré : il se déduit
des deux encodages de la tâche.

``X-User-Defined`` reste accepté pour les tâches existantes, mais n'est plus
proposé.

Encodages ASCII
---------------

.. list-table:: Unicode
   :header-rows: 1
   :widths: 30 70

   * - Nom
     - Usage
   * - ``UTF-8``
     - Unicode sur 8 bits
   * - ``UTF-8 BOM``
     - Unicode sur 8 bits, avec marque d'ordre des octets
   * - ``UTF-16BE``
     - Unicode sur 16 bits, octet de poids fort en tête
   * - ``UTF-16LE``
     - Unicode sur 16 bits, octet de poids faible en tête
   * - ``UTF-16BE BOM``
     - Unicode sur 16 bits, octet de poids fort en tête, avec marque d'ordre des octets
   * - ``UTF-16LE BOM``
     - Unicode sur 16 bits, octet de poids faible en tête, avec marque d'ordre des octets
   * - ``UTF-32BE``
     - Unicode sur 32 bits, octet de poids fort en tête
   * - ``UTF-32LE``
     - Unicode sur 32 bits, octet de poids faible en tête
   * - ``UTF-32BE BOM``
     - Unicode sur 32 bits, octet de poids fort en tête, avec marque d'ordre des octets
   * - ``UTF-32LE BOM``
     - Unicode sur 32 bits, octet de poids faible en tête, avec marque d'ordre des octets

.. list-table:: ISO 8859
   :header-rows: 1
   :widths: 30 70

   * - Nom
     - Usage
   * - ``ISO 8859-1``
     - Europe de l'Ouest (Latin-1)
   * - ``ISO 8859-2``
     - Europe centrale (Latin-2)
   * - ``ISO 8859-3``
     - Europe du Sud (Latin-3)
   * - ``ISO 8859-4``
     - Europe du Nord (Latin-4)
   * - ``ISO 8859-5``
     - Cyrillique
   * - ``ISO 8859-6``
     - Arabe
   * - ``ISO-8859-6E``
     - Arabe, direction du texte explicite
   * - ``ISO-8859-6I``
     - Arabe, direction du texte implicite
   * - ``ISO 8859-7``
     - Grec
   * - ``ISO 8859-8``
     - Hébreu
   * - ``ISO-8859-8E``
     - Hébreu, direction du texte explicite
   * - ``ISO-8859-8I``
     - Hébreu, direction du texte implicite
   * - ``ISO 8859-9``
     - Turc (Latin-5)
   * - ``ISO 8859-10``
     - Langues nordiques (Latin-6)
   * - ``ISO 8859-13``
     - Pays baltes (Latin-7)
   * - ``ISO 8859-14``
     - Langues celtiques (Latin-8)
   * - ``ISO 8859-15``
     - Europe de l'Ouest, avec euro (Latin-9)
   * - ``ISO 8859-16``
     - Europe du Sud-Est (Latin-10)

.. list-table:: Windows
   :header-rows: 1
   :widths: 30 70

   * - Nom
     - Usage
   * - ``Windows 874``
     - Windows, thaï
   * - ``Windows 1250``
     - Windows, Europe centrale
   * - ``Windows 1251``
     - Windows, cyrillique
   * - ``Windows 1252``
     - Windows, Europe de l'Ouest
   * - ``Windows 1253``
     - Windows, grec
   * - ``Windows 1254``
     - Windows, turc
   * - ``Windows 1255``
     - Windows, hébreu
   * - ``Windows 1256``
     - Windows, arabe
   * - ``Windows 1257``
     - Windows, pays baltes
   * - ``Windows 1258``
     - Windows, vietnamien

.. list-table:: PC (DOS), Mac et autres
   :header-rows: 1
   :widths: 30 70

   * - Nom
     - Usage
   * - ``IBM Code Page 437``
     - PC (DOS), États-Unis
   * - ``IBM Code Page 850``
     - PC (DOS), Europe de l'Ouest (Latin-1)
   * - ``IBM Code Page 852``
     - PC (DOS), Europe centrale (Latin-2)
   * - ``IBM Code Page 855``
     - PC (DOS), cyrillique
   * - ``Windows Code Page 858``
     - PC (DOS), Europe de l'Ouest (Latin-1), avec euro
   * - ``IBM Code Page 860``
     - PC (DOS), Portugal
   * - ``IBM Code Page 862``
     - PC (DOS), hébreu
   * - ``IBM Code Page 863``
     - PC (DOS), Canada
   * - ``IBM Code Page 865``
     - PC (DOS), Danemark, Norvège
   * - ``IBM Code Page 866``
     - PC (DOS), cyrillique
   * - ``KOI8-R``
     - Cyrillique, russe
   * - ``KOI8-U``
     - Cyrillique, ukrainien
   * - ``Macintosh``
     - Mac OS, Europe de l'Ouest
   * - ``Macintosh Cyrillic``
     - Mac OS, cyrillique

.. list-table:: Langues asiatiques (multi-octets)
   :header-rows: 1
   :widths: 30 70

   * - Nom
     - Usage
   * - ``EUC-JP``
     - Japonais (Unix)
   * - ``ISO-2022-JP``
     - Japonais (messagerie)
   * - ``Shift JIS``
     - Japonais (Windows)
   * - ``EUC-KR``
     - Coréen
   * - ``GB18030``
     - Chinois (norme nationale complète)
   * - ``GBK``
     - Chinois simplifié
   * - ``HZ-GB2312``
     - Chinois simplifié (messagerie)
   * - ``Big5``
     - Chinois traditionnel

Encodages EBCDIC
----------------

.. list-table:: Alphabet latin
   :header-rows: 1
   :widths: 30 70

   * - Nom
     - Usage
   * - ``IBM Code Page 037``
     - États-Unis, Canada, Pays-Bas, Portugal, Brésil, Australie, Nouvelle-Zélande
   * - ``IBM Code Page 038``
     - International, sans lettres accentuées
   * - ``IBM Code Page 256``
     - Pays-Bas
   * - ``IBM Code Page 273``
     - Allemagne, Autriche
   * - ``IBM Code Page 274``
     - Belgique
   * - ``IBM Code Page 275``
     - Brésil
   * - ``IBM Code Page 277``
     - Danemark, Norvège
   * - ``IBM Code Page 278``
     - Finlande, Suède
   * - ``IBM Code Page 280``
     - Italie
   * - ``IBM Code Page 281``
     - Japon, alphabet latin
   * - ``IBM Code Page 284``
     - Espagne, Amérique latine
   * - ``IBM Code Page 285``
     - Royaume-Uni
   * - ``IBM Code Page 297``
     - France
   * - ``IBM Code Page 500``
     - International (Belgique, Canada, Suisse), Latin-1
   * - ``IBM Code Page 870``
     - Europe centrale (Latin-2)
   * - ``IBM Code Page 871``
     - Islande
   * - ``IBM Code Page 905``
     - Turc (Latin-3)
   * - ``IBM Code Page 1026``
     - Turc (Latin-5)
   * - ``IBM Code Page 1047``
     - Latin-1, systèmes ouverts (z/OS UNIX)
   * - ``IBM Code Page 1112``
     - Pays baltes
   * - ``IBM Code Page 1122``
     - Estonie
   * - ``IBM Code Page 1130``
     - Vietnamien
   * - ``IBM Code Page 1140``
     - États-Unis, Canada, Pays-Bas, Portugal, Brésil, Australie, Nouvelle-Zélande, avec euro
   * - ``IBM Code Page 1141``
     - Allemagne, Autriche, avec euro
   * - ``IBM Code Page 1142``
     - Danemark, Norvège, avec euro
   * - ``IBM Code Page 1143``
     - Finlande, Suède, avec euro
   * - ``IBM Code Page 1144``
     - Italie, avec euro
   * - ``IBM Code Page 1145``
     - Espagne, Amérique latine, avec euro
   * - ``IBM Code Page 1146``
     - Royaume-Uni, avec euro
   * - ``IBM Code Page 1147``
     - France, avec euro
   * - ``IBM Code Page 1148``
     - International (Belgique, Canada, Suisse), Latin-1, avec euro
   * - ``IBM Code Page 1149``
     - Islande, avec euro
   * - ``IBM Code Page 1153``
     - Europe centrale (Latin-2), avec euro
   * - ``IBM Code Page 1155``
     - Turc (Latin-5), avec euro
   * - ``IBM Code Page 1156``
     - Pays baltes, avec euro
   * - ``IBM Code Page 1157``
     - Estonie, avec euro
   * - ``IBM Code Page 1164``
     - Vietnamien, avec euro

.. list-table:: Autres alphabets
   :header-rows: 1
   :widths: 30 70

   * - Nom
     - Usage
   * - ``IBM Code Page 290``
     - Japonais, katakana
   * - ``IBM Code Page 420``
     - Arabe
   * - ``IBM Code Page 423``
     - Grèce, avec lettres latines accentuées
   * - ``IBM Code Page 424``
     - Hébreu
   * - ``IBM Code Page 803``
     - Hébreu, jeu réduit
   * - ``IBM Code Page 875``
     - Grèce
   * - ``IBM Code Page 880``
     - Cyrillique multilingue, jeu réduit
   * - ``IBM Code Page 918``
     - Ourdou
   * - ``IBM Code Page 1025``
     - Cyrillique multilingue
   * - ``IBM Code Page 1097``
     - Persan (farsi)
   * - ``IBM Code Page 1123``
     - Cyrillique, Ukraine
   * - ``IBM Code Page 1132``
     - Lao
   * - ``IBM Code Page 1137``
     - Dévanagari (hindi)
   * - ``IBM Code Page 1154``
     - Cyrillique multilingue, avec euro
   * - ``IBM Code Page 1158``
     - Cyrillique, Ukraine, avec euro
   * - ``IBM Code Page 1160``
     - Thaï, avec euro
   * - ``IBM Code Page 1166``
     - Cyrillique, Kazakhstan, avec euro
   * - ``IBM Code Page 4517``
     - Arabe, avec lettres latines accentuées
   * - ``IBM Code Page 4899``
     - Hébreu, jeu réduit, avec euro
   * - ``IBM Code Page 4971``
     - Grèce, avec euro
   * - ``IBM Code Page 9030``
     - Thaï étendu
   * - ``IBM Code Page 12712``
     - Hébreu, avec euro
   * - ``IBM Code Page 16804``
     - Arabe, avec euro
