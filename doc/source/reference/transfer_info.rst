.. _reference-transfer-info:

Clés réservées des informations de transfert
============================================

Les :term:`infos de transfert` sont des paires clé-valeur libres, attachées à un
transfert. Waarp Gateway donne un sens aux clés ci-dessous, de la forme
``__clé__``. Certaines sont posées par l'exploitant (dans la commande qui crée
le transfert, dans une tâche ``TRANSFER`` ou ``PREREGISTER``, par la tâche
``SETINFO``) et lues par la Gateway ; d'autres sont écrites par la Gateway
pendant le transfert ; quelques-unes sont posées d'un côté du transfert et
écrites de l'autre.

Chaque clé se lit dans un traitement par le marqueur ``#TI___clé__#``
(:ref:`reference-tasks-substitutions`). L'interface web les propose dans la
liste des variables, par famille, et sur le nom d'une information de transfert.

Transfert
---------

.. list-table::
   :header-rows: 1
   :widths: 30 14 56

   * - Clé
     - Posée par
     - Signification
   * - ``__followID__``
     - Les deux
     - Identifiant de flux (R66) ; relie aussi un transfert à son acquittement
       (voir :doc:`protocols/pesit`).
   * - ``__userContent__``
     - Les deux
     - Contenu utilisateur (R66).
   * - ``__syncTransferID__``
     - Gateway
     - Transfert parent d'un transfert synchronisé.
   * - ``__syncTransferRank__``
     - Gateway
     - Rang d'un transfert synchronisé parmi ceux de son parent.

PeSIT
-----

Le détail est dans la page du protocole :doc:`protocols/pesit`.

.. list-table::
   :header-rows: 1
   :widths: 30 14 56

   * - Clé
     - Posée par
     - Signification
   * - ``__clientConnFreetext__``
     - Les deux
     - Texte libre envoyé par le client à la connexion : lu par la Gateway
       cliente, écrit par la Gateway serveur.
   * - ``__clientTransFreetext__``
     - Les deux
     - Texte libre envoyé par le client à la sélection du fichier.
   * - ``__clientTransFreetextRaw__``
     - Gateway
     - Texte libre du client reçu tel quel (PI 99) quand le profil historique
       place le texte libre dans le PI 94.
   * - ``__serverConnFreetext__``
     - Gateway
     - Texte libre envoyé par le serveur à la connexion, écrit par la Gateway
       cliente.
   * - ``__serverTransFreetext__``
     - Les deux
     - Texte libre envoyé par le serveur à la sélection du fichier : lu par la
       Gateway serveur, écrit par la Gateway cliente.
   * - ``__fileEncoding__``
     - Les deux
     - Encodage du fichier (PI 16) : lu à l'émission, écrit à la réception.
   * - ``__fileType__``
     - Les deux
     - Type de fichier (PI 11).
   * - ``__organization__``
     - Les deux
     - Organisation du fichier (PI 33).
   * - ``__customerID__``
     - Les deux
     - Identifiant de client (PI 61).
   * - ``__bankID__``
     - Les deux
     - Identifiant de banque (PI 62).
   * - ``__articlesFormat__``
     - Les deux
     - Format des articles, fixe ou variable (PI 31).
   * - ``__articlesLengths__``
     - Les deux
     - Longueurs des articles, sous forme compacte (longueur et nombre
       d'occurrences à la suite).
   * - ``__articlesSeparator__``
     - Exploitant
     - ``LF`` ou ``CRLF`` : à l'émission, un enregistrement du fichier texte par
       article, séparateur exclu.

Acquittements
-------------

Les acquittements sont décrits avec la tâche :doc:`tasks/sendmessage` et dans la
page du protocole :doc:`protocols/pesit`.

.. list-table::
   :header-rows: 1
   :widths: 30 14 56

   * - Clé
     - Posée par
     - Signification
   * - ``__ackExpected__``
     - Les deux
     - ``true`` : le transfert attend un acquittement ; posée à la main ou par
       l'option ``expectsAck`` du partenaire PeSIT.
   * - ``__ackWaitTimeout__``
     - Les deux
     - Durée d'attente de l'acquittement (``30m``) ; posée à la main ou par
       l'option ``ackTimeout`` du partenaire ; 10 minutes sans elle.
   * - ``__ackWaitSince__``
     - Gateway
     - Date depuis laquelle le transfert attend l'acquittement.
   * - ``__ackReceived__``
     - Gateway
     - ``true`` : l'acquittement a été reçu.
   * - ``__ackReceivedOn__``
     - Gateway
     - Date de réception de l'acquittement (ISO 8601).
   * - ``__ackSent__``
     - Gateway
     - ``true`` : la Gateway a envoyé l'acquittement (tâche ``SENDMESSAGE``).
   * - ``__ackSentOn__``
     - Gateway
     - Date d'envoi de l'acquittement (ISO 8601).
   * - ``__ackSentTo__``
     - Gateway
     - Partenaire auquel l'acquittement a été envoyé.
   * - ``__ackSentAs__``
     - Gateway
     - Compte distant utilisé pour l'envoi de l'acquittement.
