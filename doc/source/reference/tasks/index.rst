.. _reference-tasks:

###########
Traitements
###########

Lors de l'ajout d'une règle, les traitements de la règle doivent être fournis
avec leurs arguments sous forme d'un objet JSON. Cet objet JSON contient 2
attributs:

* ``type`` (*string*) - Le type de traitement (voir liste ci-dessous).
* ``args`` (*object*) - Les arguments du traitement en format JSON. La structure
  de cet objet JSON dépend du type du traitement.

**Exemple**

.. code-block:: json

   {
     "type": "COPY",
     "args": {
       "path": "/backup"
     }
   }

.. _reference-tasks-list:

Liste des traitements
=====================

.. toctree::
   :maxdepth: 1

   copy
   copyrename
   delete
   exec
   execmove
   execoutput
   move
   moverename
   rename
   transfer
   transcode
   archive
   extract
   icap
   email
   encrypt
   decrypt
   sign
   verify
   encrypt_sign
   decrypt_verify



.. _reference-tasks-substitutions:

Substitutions
=============

Les valeurs fournies dans l'objet ``args`` peuvent contenir des substitutions.

Les marqueurs de substitutions sont délimités par des signes dièse (``#``), et
sont valorisés au moment de l'exécution du traitement par les données
correspondant au transfert.

Les substitutions disponibles sont les suivantes :

======================= =============
Marqueur                Signification
======================= =============
``#TRUEFULLPATH#``      Le chemin réel du fichier sur le disque
``#TRUEFILENAME#``      Le nom réel du fichier sur le disque
``#BASEFILENAME#``      Le nom du fichier sur disque (sans extension)
``#FILEEXTENSION#``     L'extension du fichier (avec le point inclus, ex: ``.txt``)
``#ORIGINALFULLPATH#``  Le chemin d'origine du fichier avant le transfert
``#ORIGINALFILENAME#``  Le nom d'origine du fichier avant le transfert
``#FILESIZE#``          La taille du fichier
``#HOMEPATH#``          Le dossier racine de la Gateway. Ce chemin est toujours
                        absolu.
``#INPATH#``            Le dossier de réception par défaut définit dans le fichier
                        de configuration. Ce chemin est toujours absolu.
``#OUTPATH#``           Le dossier d'envoi par défaut définit dans le fichier de
                        configuration. Ce chemin est toujours absolu.
``#WORKPATH#``          Le dossier temporaire de réception par défaut définit dans
                        le fichier de configuration. Ce chemin est toujours absolu.
``#RULE#``              La règle utilisée par le transfert
``#DATE#``              La date (au format ``AAAAMMJJ``) au moment de l'exécution
                        de la tâche
``#HOUR#``              L'heure (au format ``HHMMSS``) au moment de l'exécution
                        de la tâche
``#TIMESTAMP(format)#`` Un timestamp au format personnalisable. Le format est
                        constitué d'une suite de token qui seront remplacés par
                        leur valeur correspondante. La table de correspondance
                        peut être consultée :ref:`ci-dessous <ref-timestamp-format>`.
                        Par défaut, le format ``YYYY-MM-DD_HHmmss`` est utilisé.
``#REMOTEHOST#``        L'identifiant du partenaire distant
``#LOCALHOST#``         L'identifiant du partenaire local
``#TRANSFERID#``        L'identifiant du transfert
``#REQUESTERHOST#``     L'identifiant du partenaire qui a demandé le transfert
``#REQUESTEDHOST#``     L'identifiant du partenaire qui a reçu la demande de
                        transfert
``#FULLTRANSFERID#``    Un identifiant "étendu" pour le transfert (de la forme
                        ``identifiantDeTransfert_idClient_idServeur``)
``#ERRORMSG#``          Message d'erreur (dans les traitements d'erreur)
``#ERRORCODE#``         Code d'erreur (dans les traitements d'erreur)
======================= =============

En plus de ces marqueurs standards, il est également possible de référencer les
:term:`infos de transfert` dans la définition d'un traitement. Pour ce faire,
le marqueur à utiliser est le suivant:

``#TI_<nom_de_clé>#`` où ``<nom_de_clé>`` est remplacée par le nom de la clé souhaitée.

À l'exécution, ce marqueur sera alors substitué par la valeur associée à la clé
renseignée.

Ces valeurs de substitutions sont également disponibles pour les programmes externes
appelés par les tâches EXEC sous forme de variables d'environnement. Ces variables
d'environnement ont exactement le même nom que leurs variables de substitution
correspondantes (ex: ``#TRUEFULLPATH#``).

.. ``#ARCHPATH#``
   ``#REMOTEHOSTIP#``
   ``#LOCALIP#`` 
   ``#RANKTRANSFER#`` 
   ``#BLOCKSIZE#`` 
   ``#ERRORSTRCODE#`` mauvaise définition
   ``#NOWAIT#`` 
   ``#LOCALEXEC#`` 
   définition de LOCALHOST et de REMOTEHOST ?

.. _ref-timestamp-format:

Formatage des *timestamps*
==========================

La table suivante indique les correspondance entre les différents tokens constituant
un format de *timestamp*, et leurs valeurs de remplacement une fois résolus.

À noter que tout caractère d'un format ne faisant pas partie d'un token sera
laissé inchangé. Par exemple, si un format de timestamp contient le caractère
*underscore* (``_``), celui-ci ne correspondant à aucun token dans la liste
ci-dessous, il sera donc laissé tel quel dans le timestamp final.

.. table::
   :widths: 40 10 20

   +------------------------+-------+------------------+
   | Unité de temps         | Token | Valeur           |
   +========================+=======+==================+
   | **Année**              | YYYY  | 2025             |
   |                        +-------+------------------+
   |                        | YY    | 25               |
   +------------------------+-------+------------------+
   | **Mois**               | MMMM  | January          |
   |                        +-------+------------------+
   |                        | MMM   | Jan              |
   |                        +-------+------------------+
   |                        | MM    | 01..12           |
   |                        +-------+------------------+
   |                        | M     | 1..12            |
   +------------------------+-------+------------------+
   | **Jour**               | DD    | 01..31           |
   |                        +-------+------------------+
   |                        | D     | 1..31            |
   +------------------------+-------+------------------+
   | **Jour de la semaine** | dddd  | Monday           |
   |                        +-------+------------------+
   |                        | ddd   | Mon              |
   +------------------------+-------+------------------+
   | **AM/PM**              | PM    | AM/PM            |
   |                        +-------+------------------+
   |                        | pm    | am/pm            |
   +------------------------+-------+------------------+
   | **Heure**              | HH    | 00..23           |
   |                        +-------+------------------+
   |                        | hh    | 01..12           |
   |                        +-------+------------------+
   |                        | h     | 1..12            |
   +------------------------+-------+------------------+
   | **Minutes**            | mm    | 00..59           |
   |                        +-------+------------------+
   |                        | m     | 0..59            |
   +------------------------+-------+------------------+
   | **Secondes**           | ss    | 00..59           |
   |                        +-------+------------------+
   |                        | s     | 0..59            |
   +------------------------+-------+------------------+
   | **Fuseau horaire**     | tz    | UTC, MST, CET... |
   |                        +-------+------------------+
   |                        | zz    | -06:00 .. +06:00 |
   |                        +-------+------------------+
   |                        | z     | -0600 .. +0600   |
   +------------------------+-------+------------------+

