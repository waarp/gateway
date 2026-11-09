REFORMAT
========

Le traitement ``REFORMAT`` change la mise en forme du fichier cible du
transfert, en lignes ou en enregistrements, sans toucher à son contenu. Les
arguments sont :

* ``charset`` (*string*) - L'encodage du fichier, désigné comme pour
  :doc:`transcode`, qui en donne :ref:`la liste <transcode-encodages>`.
* ``records`` (*string*) - Le découpage du fichier : ``lines`` des lignes,
  ``fixed`` des enregistrements de longueur fixe, ``rdw`` des enregistrements
  précédés de leur longueur (RDW), comme z/OS les écrit.
* ``recordLength`` (*int*) - Avec ``fixed`` : la longueur des enregistrements,
  en octets, de 1 à 32760.
* ``trimPadding`` (*bool*) - Avec ``fixed`` : retire les espaces qui terminent
  chaque enregistrement.
* ``outputRecords`` (*string*) - La mise en forme du résultat : ``none`` sans
  rien entre les enregistrements, ``lines``, ``fixed`` ou ``rdw``.
* ``outputSeparator`` (*string*) - Avec ``lines`` : la fin de ligne écrite,
  ``LF`` (par défaut), ``CRLF`` ou ``NEL`` (fichier EBCDIC seulement).
* ``outputLength`` (*int*) - Avec ``fixed`` : la longueur des enregistrements
  du résultat, en octets, de 1 à 32760 ; un enregistrement plus court est
  complété par des espaces.
* ``truncate`` (*bool*) - Avec ``fixed`` ou ``rdw`` : coupe un enregistrement
  trop long au lieu d'échouer.

``charset``, ``records`` et ``outputRecords`` sont obligatoires. Une
combinaison d'arguments qui ne s'applique pas (``recordLength`` sans
``fixed``, ``outputSeparator`` sans ``lines``…) est refusée à l'enregistrement
de la règle.

.. code-block:: yaml

   post:
     - type: REFORMAT
       args:
         charset: "IBM Code Page 297"
         records: "fixed"
         recordLength: "80"
         trimPadding: "true"
         outputRecords: "lines"

Ce que le traitement change
---------------------------

Le traitement ne change que l'enveloppe des enregistrements : les séparateurs,
les espaces de remplissage et les en-têtes de longueur (RDW). L'encodage ne lui
sert qu'à reconnaître et à écrire ces octets : la fin de ligne LF vaut ``0x25``
et l'espace ``0x40`` en EBCDIC, et une fin de ligne tient sur deux octets en
UTF-16.

Les fins de ligne suivent les règles de :doc:`transcode_reformat`. Côté EBCDIC,
``LF`` et ``CRLF`` s'écrivent ``0x0A`` et ``0x0D 0x0A``, là où le module PeSIT
de la passerelle découpe un fichier texte en articles. Un fichier EBCDIC a de
plus ses lignes reconnues sur ``0x25`` et ``0x15`` ; ``NEL`` écrit ``0x15``.

Ce que le traitement ne change pas
----------------------------------

Le contenu de chaque enregistrement est recopié octet pour octet, sans être
décodé : le traitement ne réordonne ni ne modifie les champs d'un
enregistrement. Il ne vérifie pas non plus les caractères : un octet qui ne
représente aucun caractère de l'encodage, comme une séquence UTF-8 invalide ou
un octet sans caractère dans une page de codes, passe tel quel. Pour vérifier
ou convertir les caractères, utiliser :doc:`transcode` ou
:doc:`transcode_reformat`.

Un enregistrement trop long
---------------------------

Un enregistrement plus long que ne le permet le résultat (``fixed`` ou ``rdw``)
fait échouer le traitement, et le fichier reste tel qu'il était.

Avec ``truncate``, l'enregistrement est coupé entre deux caractères, d'après
les octets :

* en UTF-8, avant le premier octet du caractère qui dépasse ;
* en UTF-16, jamais au milieu d'une paire de substitution ;
* dans une page à un octet et en UTF-32, à la longueur demandée.

Les encodages asiatiques multi-octets (Shift JIS, GBK, Big5…) ne permettent
pas de retrouver la frontière d'un caractère depuis une position :
``truncate`` y est refusé à l'enregistrement de la règle.

Changer aussi l'encodage
------------------------

Pour changer à la fois l'encodage et la mise en forme, :doc:`transcode_reformat`
le fait en une seule lecture du fichier. Enchaîner ``TRANSCODE`` et
``REFORMAT`` donne le même résultat, mais lit et écrit le fichier deux fois.
