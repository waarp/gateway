TRANSCODE&REFORMAT
==================

Le traitement ``TRANSCODE&REFORMAT`` combine les traitements :doc:`transcode`
et :doc:`reformat` : il change l'encodage textuel du fichier cible du transfert
et sa mise en forme en lignes ou en enregistrements, en une seule lecture du
fichier. Les arguments sont :

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

Les encodages se désignent comme pour :doc:`transcode`, qui en donne
:ref:`la liste <transcode-encodages>`. Les deux encodages doivent différer.

Sans ``records``, le fichier est converti en entier, comme le fait
``TRANSCODE``.

Avec ``records``, seul le contenu des enregistrements est converti : un
séparateur ou un en-tête de longueur n'est jamais pris pour du texte. Une
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
     - type: TRANSCODE&REFORMAT
       args:
         fromCharset: "ISO 8859-1"
         toCharset: "IBM Code Page 297"
         records: "lines"
