==================
Ajouter un serveur
==================

.. program:: waarp-gateway server add

Ajoute un nouveau serveur de transfert à la gateway avec les attributs fournis.

**Commande**

.. code-block:: shell

   waarp-gateway server add

**Options**

.. option:: -n <NAME>, --name=<NAME>

   Le nom du serveur. Doit être unique.

.. option:: -p <PROTO>, --protocol=<PROTO>

   Le protocole utilisé par le serveur.

.. option:: -a <ADDRESS>, --address=<ADDRESS>

   L'adresse du serveur (au format [adresse:port]).

.. option:: --root-dir=<ROOT_DIR>

   Le dossier racine du serveur. Peut être un chemin relatif à la racine de la
   Waarp Gateway, ou bien absolu.

.. option:: --receive-dir=<RECV_DIR>

   Le dossier de réception du serveur. Les fichiers reçus sur le serveur seront
   déposés dans ce dossier (sauf si la règle de transfert supplante ce dossier
   avec son propre dossier local). Peut être un chemin relatif au ``root-dir``
   du serveur, ou bien absolu.

.. option:: --send-dir=<SEND_DIR>

   Le dossier d'envoi du serveur. Les fichiers téléchargés depuis le serveur
   seront récupérés dans ce dossier (sauf si la règle de transfert supplante ce
   dossier avec son propre dossier local). Peut être un chemin relatif au
   ``root-dir`` du serveur, ou bien absolu.

.. option:: --tmp-dir=<TMP_DIR>

   Le dossier de réception temporaire du serveur. Les fichiers en cours de
   réception par le serveur seront écrits dans ce dossier pour la durée du
   transfert (sauf si la règle de transfert supplante ce dossier avec son propre
   dossier temporaire local) avant d'être déposés dans le dossier de réception
   une fois le transfert terminé. Peut être un chemin relatif au ``root-dir``
   du serveur, ou bien absolu.

.. option:: -c <KEY:VAL>, --config=<KEY:VAL>

   La configuration protocolaire du serveur. Répéter pour chaque paramètre de la
   configuration. Les options de la configuration varient en fonction du protocole
   utilisé (voir :ref:`configuration protocolaire <reference-proto-config>` pour
   plus de détails).

.. option:: -r <ROOT>, --root=<ROOT>

   .. deprecated:: 0.5.0

      Remplacé par l'option ``--root-dir``.

   Le dossier racine du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine de Waarp Gateway renseignée
   dans le fichier de configuration.

.. option:: -i <IN_DIR>, --in=<IN_DIR>

   .. deprecated:: 0.5.0

      Remplacé par l'option ``--receive-dir``.

   Le dossier de réception du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine du serveur.

.. option:: -o <OUT_DIR>, --out=<OUT_DIR>

   .. deprecated:: 0.5.0

      Remplacé par l'option ``--send-dir``.

   Le dossier d'envoi du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine du serveur.

.. option:: -w <WORK_DIR>, --work=<WORK_DIR>

   .. deprecated:: 0.5.0

      Remplacé par l'option ``--tmp-dir``.


**Exemple**

.. code-block:: shell

   waarp-gateway server add -n 'gw_r66' -p 'r66-tls' -a 'localhost:6666' --root-dir 'r66/root' --config 'blockSize:32768'
