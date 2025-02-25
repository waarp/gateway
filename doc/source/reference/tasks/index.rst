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


.. _reference-tasks-substitutions:

Substitutions
=============

Les valeurs fournies dans l'objet ``args`` peuvent contenir des substitutions.

Les marqueurs de substitutions sont délimités par des signes dièse (``#``), et
sont valorisés au moment de l'exécution du traitement par les données
correspondant au transfert.

Les substitutions disponibles sont les suivantes :

====================== =============
Marqueur               Signification
====================== =============
``#TRUEFULLPATH#``     Le chemin réel du fichier sur le disque
``#TRUEFILENAME#``     Le nom réel du fichier sur le disque
``#ORIGINALFULLPATH#`` Le chemin d'origine du fichier avant le transfert
``#ORIGINALFILENAME#`` Le nom d'origine du fichier avant le transfert
``#FILESIZE#``         La taille du fichier
``#HOMEPATH#``         Le dossier racine de la Gateway. Ce chemin est toujours
                       absolu.
``#INPATH#``           Le dossier de réception par défaut définit dans le fichier
                       de configuration. Ce chemin est toujours absolu.
``#OUTPATH#``          Le dossier d'envoi par défaut définit dans le fichier de
                       configuration. Ce chemin est toujours absolu.
``#WORKPATH#``         Le dossier temporaire de réception par défaut définit dans
                       le fichier de configuration. Ce chemin est toujours absolu.
``#RULE#``             La règle utilisée par le transfert
``#DATE#``             La date (au format ``AAAAMMJJ``) au moment de l'exécution
                       de la tâche
``#HOUR#``             L'heure (au format ``HHMMSS``) au moment de l'exécution
                       de la tâche
``#REMOTEHOST#``       L'identifiant du partenaire distant
``#LOCALHOST#``        L'identifiant du partenaire local
``#TRANSFERID#``       L'identifiant du transfert
``#REQUESTERHOST#``    L'identifiant du partenaire qui a demandé le transfert
``#REQUESTEDHOST#``    L'identifiant du partenaire qui a reçu la demande de
                       transfert
``#FULLTRANSFERID#``   Un identifiant "étendu" pour le transfert (de la forme
                       ``identifiantDeTransfert_idClient_idServeur``)
``#ERRORMSG#``         Message d'erreur (dans les traitements d'erreur)
``#ERRORCODE#``        Code d'erreur (dans les traitements d'erreur)
====================== =============

En plus de ces marqueurs standards, il est également possible de référencer les
:term:`infos de transfert` dans la définition d'un traitement. Pour ce faire,
le marqueur à utiliser est le suivant:

``#TI_<nom_de_clé>#`` où ``<nom_de_clé>`` est remplacée par le nom de la clé souhaitée.

À l'exécution, ce marqueur sera alors substitué par la valeur associée à la clé
renseignée.

.. ``#ARCHPATH#``
   ``#REMOTEHOSTIP#``
   ``#LOCALIP#`` 
   ``#RANKTRANSFER#`` 
   ``#BLOCKSIZE#`` 
   ``#ERRORSTRCODE#`` mauvaise définition
   ``#NOWAIT#`` 
   ``#LOCALEXEC#`` 
   définition de LOCALHOST et de REMOTEHOST ?

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
