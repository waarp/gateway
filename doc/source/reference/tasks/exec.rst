EXEC
====

Le traitement 'EXEC' exécute un programme externe. Les arguments sont:

* **path** (*string*) - Le chemin du programme à exécuter.
* **args** (*string*) - Les arguments du programme.
* **delay** (*number*) - La durée d'exécution maximum du programme (en ms). Si
  le programme n'a pas terminé après que cette durée soit écoulée, le programme
  sera interrompu et le traitement sera considéré comme ayant échoué.