.. _proto-config-sftp:

Configuration SFTP
##################

Options communes
================

Les champs suivants sont communs aux configuration client, serveur et partenaire :

* **keyExchanges** (*array of string*) - *Optionnel* Liste des algorithmes d'échange de clé
  autorisés. Les algorithmes supportés sont (par ordre de préférence) :

  - ``curve25519-sha256@libssh.org``
  - ``ecdh-sha2-nistp256``
  - ``ecdh-sha2-nistp384``
  - ``ecdh-sha2-nistp521``
  - ``diffie-hellman-group-exchange-sha256`` (*Uniquement supporté par le client*)
  - ``diffie-hellman-group1-sha1`` [Déprécié]
  - ``diffie-hellman-group14-sha1`` [Déprécié]

  |

* **ciphers** (*array of string*) - *Optionnel* Liste des algorithmes de cryptage symétrique 
  de données autorisés sur le serveur. Les algorithmes supportés sont (par ordre de
  préférence) :

  - ``aes128-gcm@openssh.com``
  - ``chacha20-poly1305@openssh.com``
  - ``aes128-ctr``
  - ``aes192-ctr``
  - ``aes256-ctr``

  |

  Les algorithmes suivants sont également supportés mais ne sont pas activés
  par défaut :

  - ``arcfour256``
  - ``arcfour128``
  - ``arcfour``
  - ``aes128-cbc``
  - ``3des-cbc``

  |

* **macs** (*array of string*) -  *Optionnel* Liste des algorithmes d'authentification de message 
  (MAC) autorisés sur le serveur. Les algorithmes supportés sont (par ordre de préférence) :

  - ``hmac-sha2-256-etm@openssh.com``
  - ``hmac-sha2-256``
  - ``hmac-sha1`` [Déprécié]
  - ``hmac-sha1-96`` [Déprécié]

  |

  Par défaut, tous les algorithmes sont autorisés.

  |

Configuration client
====================

En dehors des options communes spécifiées ci-dessus, la configuration client
SFTP ne comporte pas d'options spécifiques.

Configuration serveur
=====================


En dehors des options communes spécifiées ci-dessus, la configuration serveur
SFTP ne comporte pas d'options spécifiques.

Configuration partenaire
========================

Les champs suivants ne sont disponibles que pour la configuration partenaire :

* **useStat** (*boolean*) - *Optionnel* Lorsque la *gateway* récupère un fichier
  depuis un serveur SFTP distant, le SFTP client de la *gateway* envoie une
  commande SFTP ``Fstat`` sur le fichier ouvert pour récupérer sa taille avant de
  commencer à le lire. Cependant, la commande ``Fstat`` peut causer des problèmes
  de compatibilité avec certains serveurs SFTP. Cette option permet de remplacer
  la commande ``Fstat`` par une commande ``Stat`` qui est plus largement supportée.

  **Note:** Cette option n'a aucun effet si l'option ``disableClientConcurrentReads``
  décrite ci-dessous est également activée.

  |

* **disableClientConcurrentReads** (*boolean*) - *Optionnel* Par défaut, lorsque
  la *gateway* récupère un fichier depuis un serveur SFTP distant, le fichier est
  lu en parallèle par plusieurs *workers* afin de maximiser le débit de transfert.
  Cependant, cette méthode de lecture en parallèle requiert que le client connaisse
  la taille du fichier (afin de déterminer le nombre de *workers* à utiliser). Pour
  cela, le client envoie une commande ``Fstat`` (ou ``Stat``) au fichier une fois
  ouvert. Cependant, cette commande peut parfois causer des problèmes avec certains
  serveurs, notamment les serveurs ne permettant qu'une lecture unique du fichier
  (*read-once*). Cette option permet de désactiver la lecture concurrente du fichier,
  ce qui désactive également l'envoi de cette commande ``Fstat``. Notez cependant,
  que en conséquence, le fichier sera alors lu de façon séquentielle ce qui impactera
  les performances du transfert.

  **Note:** Activer cette option rend l'option ``useStat`` décrite ci-dessus ineffective.

|

**Exemple**

.. code-block:: json

   {
     "keyExchanges": [
       "diffie-hellman-group1-sha1",
       "diffie-hellman-group14-sha1",
       "ecdh-sha2-nistp256",
       "ecdh-sha2-nistp384",
       "ecdh-sha2-nistp521",
       "curve25519-sha256@libssh.org"
     ],
     "ciphers": [
       "aes128-gcm@openssh.com",
       "aes128-ctr",
       "aes192-ctr",
       "aes256-ctr",
       "chacha20-poly1305@openssh.com"
     ],
     "macs": [
       "hmac-sha2-256-etm@openssh.com",
       "hmac-sha2-256",
       "hmac-sha1",
       "hmac-sha1-96"
     ],
     "useStat": true,
     "disableClientConcurrentReads": false
   }
