.. _ref-task-sendmessage:

SENDMESSAGE
===========

Le traitement ``SENDMESSAGE`` envoie un message PeSIT (F.MESSAGE) a un
partenaire distant. Il est principalement utilise pour les acquittements
Store-and-Forward.

La tache ne fonctionne qu'avec les partenaires utilisant le protocole PeSIT
ou PeSIT-TLS. Elle ouvre une connexion PeSIT dediee, envoie le message, puis
ferme la connexion.

La tache peut etre conditionnelle : si le parametre ``condition`` est
renseigne, la tache verifie que la cle TransferInfo correspondante existe
et vaut "1". Sinon, la tache est silencieusement ignoree.

Parametres
----------

* ``partner`` (*string*, **obligatoire**) - Le nom du partenaire PeSIT distant
  vers lequel envoyer le message. Supporte la substitution de variables
  (ex: ``#TI___ackPartner__#``).
* ``account`` (*string*, optionnel) - Le login du compte distant a utiliser
  pour l'authentification. Si omis, le premier compte du partenaire est
  utilise.
* ``message`` (*string*, optionnel) - Le contenu du F.MESSAGE a envoyer.
  Supporte la substitution de variables (ex: ``#TRUEFILENAME#``,
  ``#TRANSFERID#``). Maximum 4096 caracteres.
* ``transferId`` (*string*, optionnel) - L'identifiant de transfert PeSIT
  a referencer dans le F.MESSAGE. Supporte la substitution de variables
  (ex: ``#TI___originalTransferID__#``).
* ``condition`` (*string*, optionnel) - Une cle TransferInfo a verifier
  avant l'envoi. Si la cle existe et sa valeur est "1", le message est
  envoye. Sinon, la tache est silencieusement ignoree. Exemple :
  ``__ackRequested__``.

Exemple
-------

Envoi d'un acquittement conditionnel apres reception d'un fichier :

.. code-block:: json

   {
     "type": "SENDMESSAGE",
     "args": {
       "partner": "#TI___ackPartner__#",
       "message": "fichier #TRUEFILENAME# livre avec succes",
       "transferId": "#TI___originalTransferID__#",
       "condition": "__ackRequested__"
     }
   }

Store-and-Forward
-----------------

Pour mettre en place un relais Store-and-Forward avec acquittement :

1. Configurez la regle de reception du relais avec une tache ``TRANSFER``
   (``copyInfo: true``, ``synchronous: true``) pour relayer le fichier.
2. Sur le destinataire final, configurez une tache ``SENDMESSAGE`` en
   post-traitement avec la condition ``__ackRequested__``.
3. L'emetteur initial doit inclure dans le TransferInfo :
   ``__ackRequested__: "1"`` et ``__ackPartner__: "nom-emetteur"``.
4. Ces cles se propagent automatiquement de hop en hop via ``copyInfo: true``.
